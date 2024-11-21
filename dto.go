package main

import (
	"time"
)

type Record struct {
	ID          string    `sql:"id"`
	TS          time.Time `sql:"ts"`
	Failure     int       `sql:"failure"`
	Description string    `sql:"description"`
}
