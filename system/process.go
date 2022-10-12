package system

// func init() {
// 	// gou.RegisterProcessHandler("yao.system.Exec", processExec)
// }

// // processExec execute the system command
// func processExec(process *gou.Process) interface{} {
// 	process.ValidateArgNums(1)
// 	cmd := process.ArgsString(0)

// 	_, err := exec.LookPath(cmd)
// 	if err != nil {
// 		exception.New("command %s not found: %s", 400, cmd, err.Error()).Throw()
// 		return nil
// 	}

// 	args := []string{}
// 	for i, arg := range process.Args {
// 		if i == 0 {
// 			continue
// 		}
// 		args = append(args, fmt.Sprintf("%v", arg))
// 	}

// 	res, err := exec.Command(cmd, args...).Output()
// 	if err != nil {
// 		exception.New("command %s error: %s", 500, cmd, err.Error()).Throw()
// 		return nil
// 	}

// 	return strings.TrimRight(string(res), "\n")
// }
