package main

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gorcon/rcon"
)

type FactorioRcon struct {
	*rcon.Conn
}

func NewFactorioRcon(server, password string) (*FactorioRcon, error) {
	rconn, err := rcon.Dial(server, password)
	if err != nil {
		return nil, err
	}
	slog.Info("Connected RCON server.")

	// Check exists "/pause" command
	{
		response, err := rconn.Execute("/help pause")
		if err != nil {
			return nil, err
		}

		if !strings.Contains(response, "/pause") || strings.Contains(response, "Unknown command") {
			return nil, errors.New("not found \"/pause\" command.\nPlease install \"pause-commands\" mod")
		}
	}

	// Check exists "/unpause" command
	{
		response, err := rconn.Execute("/help unpause")
		if err != nil {
			return nil, err
		}

		if !strings.Contains(response, "/unpause") || strings.Contains(response, "Unknown command") {
			return nil, errors.New("not found \"/unpause\" command.\nPlease install \"pause-commands\" mod")
		}
	}

	return &FactorioRcon{rconn}, nil
}

func (f *FactorioRcon) Pause() error {
	_, err := f.Execute("/pause")
	if err != nil {
		return err
	}

	return nil
}

func (f *FactorioRcon) UnPause() error {
	_, err := f.Execute("/unpause")
	if err != nil {
		return err
	}

	return nil
}

func (f *FactorioRcon) Shout(m string) error {
	_, err := f.Execute(fmt.Sprintf("/shout %s", m))
	if err != nil {
		return err
	}

	return nil
}
