package logger

import (
	"go.uber.org/zap"
)

var Logger *zap.Logger

func Init() {
	var err error
	Logger, err = zap.NewProduction() 
	if err != nil {
		panic(err)
	}
}
