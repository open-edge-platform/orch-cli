
#!/usr/bin/env bash
set -u

rerun_failed=0
for arg in "$@"; do
  case "$arg" in
    --rerun-failed)
      rerun_failed=1
      ;;
    -h|--help)
      echo "Usage: $0 [--rerun-failed]"
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg"
      echo "Usage: $0 [--rerun-failed]"
      exit 2
      ;;
  esac
done

export GOPATH="$(go env GOPATH)"
export PATH="$GOPATH/bin:$PATH"

if [ "$rerun_failed" -eq 0 ]; then
  rm -rf ./fuzz-logs
  mkdir -p fuzz-logs
  : > fuzz-logs/result.log
else
  mkdir -p fuzz-logs
  touch fuzz-logs/result.log
fi

mapfile -t all_fuzz_tests < <(go test ./internal/cli -run=^$ -list '^Fuzz' | grep '^Fuzz')
fuzz_count="${#all_fuzz_tests[@]}"
echo "Found ${fuzz_count} fuzz tests in ./internal/cli" | tee -a fuzz-logs/result.log
if [ "$fuzz_count" -eq 0 ]; then
  exit 0
fi

targets=()
if [ "$rerun_failed" -eq 1 ]; then
  shopt -s nullglob
  for f in fuzz-logs/*.exit; do
    t=$(basename "$f" .exit)
    if [ "$(cat "$f")" -ne 0 ]; then
      if printf '%s\n' "${all_fuzz_tests[@]}" | grep -qx "$t"; then
        targets+=("$t")
      else
        echo "Skipping stale failed target ${t} (not in current fuzz list)" | tee -a fuzz-logs/result.log
      fi
    fi
  done
  shopt -u nullglob

  target_count="${#targets[@]}"
  if [ "$target_count" -eq 0 ]; then
    echo "No previously failed fuzz tests found to rerun." | tee -a fuzz-logs/result.log
    exit 0
  fi
  echo "Rerunning ${target_count} previously failed fuzz tests" | tee -a fuzz-logs/result.log
else
  targets=("${all_fuzz_tests[@]}")
  target_count="$fuzz_count"
fi

printf '%s\n' "${targets[@]}" \
| xargs -I{} -P "$target_count" bash -lc '
  t="{}"
  log="fuzz-logs/${t}.log"

  cmd=(go test ./internal/cli -run=^$ -fuzz="^${t}$" -fuzztime="${FUZZ_TIME:-30m}" -count=1 -parallel=1)

  {
    printf "==> %s\n" "${t}"
    printf "CMD: GOMEMLIMIT=2GiB GOMAXPROCS=1 "
    printf "%q " "${cmd[@]}"
    printf "\n"
  } | tee -a fuzz-logs/result.log

  GOMEMLIMIT=2GiB GOMAXPROCS=1 "${cmd[@]}" >"${log}" 2>&1
  rc=$?
  echo "${rc}" > "fuzz-logs/${t}.exit"

  if [ "${rc}" -eq 0 ]; then
    echo "PASS ${t}" | tee -a fuzz-logs/result.log
  else
    echo "FAIL ${t}" | tee -a fuzz-logs/result.log
    tail -n 40 "${log}"
  fi
'

{
  echo "---- Summary ----"
  shopt -s nullglob
  for f in fuzz-logs/*.exit; do
    t=$(basename "$f" .exit)
    if [ "$(cat "$f")" -eq 0 ]; then
      echo "PASS $t"
    else
      echo "FAIL $t"
    fi
  done
  shopt -u nullglob
} | tee -a fuzz-logs/result.log
