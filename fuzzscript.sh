rm -rf ./fuzz-logs
mkdir -p fuzz-logs

fuzz_count=$(go test ./internal/cli -run=^$ -list '^Fuzz' | grep '^Fuzz' | wc -l | tr -d ' ')
echo "Found ${fuzz_count} fuzz tests in ./internal/cli" | tee -a fuzz-logs/result.log
if [ "$fuzz_count" -eq 0 ]; then
  exit 0
fi

go test ./internal/cli -run=^$ -list '^Fuzz' \
| grep '^Fuzz' \
| head -n "$fuzz_count" \
| xargs -I{} -P "$fuzz_count" bash -lc '
  t="{}"
  log="fuzz-logs/${t}.log"
  cmd=(go test ./internal/cli -run=^$ -fuzz="^${t}$" -fuzztime="${FUZZ_TIME:-30m}" -count=1)

  {
    printf "==> %s\n" "${t}"
    printf "CMD: "
    printf "%q " "${cmd[@]}"
    printf "\n"
  } | tee -a fuzz-logs/result.log

  "${cmd[@]}" >"${log}" 2>&1
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
  for f in fuzz-logs/*.exit; do
    t=$(basename "$f" .exit)
    if [ "$(cat "$f")" -eq 0 ]; then
      echo "PASS $t"
    else
      echo "FAIL $t"
    fi
  done
} | tee -a fuzz-logs/result.log