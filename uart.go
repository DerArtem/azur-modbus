package main

import (
	"bytes"
	serial2 "github.com/albenik/go-serial"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

var port serial2.Port
var errUart error
var mutex sync.Mutex
var tempStr = strings.Builder{}
var rxBuffer = strings.Builder{}
var buff = make([]byte, 4096)

var serialMode = &serial2.Mode{

	BaudRate: 9600,
	Parity:   serial2.NoParity,
	DataBits: 8,
	StopBits: serial2.OneStopBit,
}

func UartInitBasen() error {
	port, errUart = serial2.Open("/dev/ttySTM2", serialMode)
	return nil
}

//type uartBuffer struct{ bytes.Buffer }

// var respBuffer uartBuffer
var respBuffer bytes.Buffer

func sendDataToUart(txData []byte, timeout time.Duration, retryCount int) {

	if errUart == nil {
		mutex.Lock()
		defer mutex.Unlock()

		port.SetReadTimeout(10)
		port.SetWriteTimeout(-1)

		/* Retry sending if we do not receive data in time */
		for retryCnt := 0; retryCnt < retryCount; retryCnt++ {
			var bytesWritten int
			var err error

			rxBuffer.Reset()
			respBuffer.Reset()

			if retryCnt == 0 {
				log.Debugf("send uart >> %02X", txData)
			} else {
				log.Warnf("send uart retry %d >> %02X", retryCnt, txData)
			}

			//log.Infof("send uart XXX >> %02X", txData)

			/*
				for i := 0; i < len(txData); i++ {
					bytesWritten, err = port.WriteState([]byte{txData[i]})

					if bytesWritten != 1 {
						log.Errorf("UART WriteState error: %v - bytes written %v/%v", err, bytesWritten, 1)
					}

					if err != nil {
						log.Errorf("UART WriteState error: %v - bytes written %v/%v", err, bytesWritten, 1)
					}

					time.Sleep(1 * time.Millisecond)
				}
			*/

			//time.Sleep(1 * time.Millisecond)

			/* Reset the input buffer before sending a new command */
			port.ResetInputBuffer()
			//port.ResetOutputBuffer()

			bytesWritten, err = port.Write(txData)

			if bytesWritten != len(txData) || err != nil {
				log.Errorf("UART write error: %v - bytes written %v/%v", err, bytesWritten, len(txData))
				return
			}

			rxStart := time.Now()

			/* Data might come in packets - we need to reassemble them together */
			for time.Since(rxStart) < timeout {
				n, err := port.Read(buff)

				if n > 0 {
					rxBuffer.Write(buff[:n])

					tempStr.Reset()
					tempStr.Write(buff[:n])

					if buff[n-1] == 0x0d {
						//log.Infof("rece uart << %s", rxBuffer.String())
						//log.Infof("rece uart << %02X", rxBuffer.String())
						respBuffer.WriteString(rxBuffer.String())
						return
					}
				}

				if err != nil {
					/* just ignore this error - it happens when process is switched between cores */
					if err.Error() == "operating system error: interrupted system call" {
						continue
					}

					log.Errorf("port.Read Error: %v", err)
				}
			}

			return
		}
		log.Debug("UART>> timed out, giving up!")
	}
}
