package output

func DoneForTest(sw *SafeWriter) <-chan struct{} { return sw.done }
