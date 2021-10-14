package algo

import "time"

type LogBuilder struct {
	stdOut []string
	stdErr []string
	info   []string
}

func (l *LogBuilder) Reset() *LogBuilder {
	l.stdOut = make([]string, 0)
	l.stdErr = make([]string, 0)
	l.info = make([]string, 0)

	return l
}

func (l *LogBuilder) AddStdOut(str string) *LogBuilder {
	l.stdOut = append(l.stdOut, str)
	return l
}

func (l *LogBuilder) AddStdErr(str string) *LogBuilder {
	l.stdErr = append(l.stdErr, str)
	return l
}

func (l *LogBuilder) AddInfoText(str string) *LogBuilder {
	l.info = append(l.info, str)
	return l
}

func (l *LogBuilder) AddTimestamp(outPipe *[]string) *LogBuilder {
	*outPipe = append(*outPipe, "\n[ "+l.getCurTime()+" ]\n")

	return l
}

func (l *LogBuilder) GetLastStdOut() string {
	ln := len(l.stdOut)

	if ln > 0 {
		return l.stdOut[ln-1]
	}

	return ""
}

func (l *LogBuilder) GetLastStdErr() string {
	ln := len(l.stdErr)

	if ln > 0 {
		return l.stdErr[ln-1]
	}

	return ""
}

func (l *LogBuilder) GetLastInfo() string {
	ln := len(l.info)

	if ln > 0 {
		return l.info[ln-1]
	}

	return ""
}

func (l *LogBuilder) FormatLog(logArray []string) string {
	var output string

	for _, str := range logArray {
		output += str
	}

	return output
}

func (l *LogBuilder) getCurTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
