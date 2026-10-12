package env

// Java 把 JDK 目录写入 JAVA_HOME，并把 %JAVA_HOME%\bin 加进系统 PATH。
func Java(dest string) error {
	if err := setSystemString("JAVA_HOME", dest); err != nil {
		return err
	}
	return addToPath(`%JAVA_HOME%\bin`)
}
