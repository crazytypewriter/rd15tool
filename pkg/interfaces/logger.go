package interfaces

type LogWriter interface {
	LogWrite(message string)
}

type LogWriteNoNewLine interface {
	LogWrite(message string)
}

type LogWriterWithProgress interface {
	LogWriteWithProgress(startText string, task func() error)
}
