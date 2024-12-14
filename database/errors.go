package database

import "errors"

var ErrFatal = errors.New("fatal database error")
var ErrConnection = errors.New("database connection error")
var ErrNoConnectionWorker = errors.New("no connection worker is running")
