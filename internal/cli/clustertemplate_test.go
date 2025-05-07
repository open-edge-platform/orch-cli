package cli

import "fmt"

func (s *CLITestSuite) listClusterTemplates(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list clustertemplates --project %s`,
		publisher))
	return s.runCommand(commandString)
}
