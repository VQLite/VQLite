package utils

func SliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

//func RestartSelf() error {
//	self, err := osext.Executable()
//
//	self, err := os.Executable()
//
//	if err != nil {
//		return err
//	}
//	args := os.Args
//	env := os.Environ()
//	// Windows does not support exec syscall.
//	if runtime.GOOS == "windows" {
//		cmd := exec.Command(self, args[1:]...)
//		cmd.Stdout = os.Stdout
//		cmd.Stderr = os.Stderr
//		cmd.Stdin = os.Stdin
//		cmd.Env = env
//		err := cmd.Run()
//		if err == nil {
//			os.Exit(0)
//		}
//		return err
//	}
//	return syscall.Exec(self, args, env)
//}
