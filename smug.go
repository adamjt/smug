package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return path
		}

		return strings.Replace(path, "~", userHome, 1)
	}

	return path
}

func Contains(slice []string, s string) bool {
	for _, e := range slice {
		if e == s {
			return true
		}
	}

	return false
}

type Smug struct {
	tmux      Tmux
	commander Commander
}

func (smug Smug) execShellCommands(commands []string, path string) error {
	for _, c := range commands {
		args := strings.Split(c, " ")

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = path

		_, err := smug.commander.Exec(cmd)
		if err != nil {
			return &ShellError{c, err}
		}
	}
	return nil
}

func (smug Smug) StopSession(config Config) error {
	sessionRoot := ExpandPath(config.Root)

	err := smug.execShellCommands(config.Stop, sessionRoot)
	if err != nil {
		return err
	}

	_, err = smug.tmux.StopSession(config.Session)

	return err
}

func (smug Smug) StartSession(config Config, windows []string) error {
	var ses string
	var err error

	sessionRoot := ExpandPath(config.Root)

	sessionExists := smug.tmux.SessionExists(config.Session)
	if !sessionExists {
		err = smug.execShellCommands(config.BeforeStart, sessionRoot)
		if err != nil {
			return err
		}

		ses, err = smug.tmux.NewSession(config.Session)
		if err != nil {
			return err
		}
	} else {
		ses = config.Session + ":"
	}

	for _, w := range config.Windows {
		if (len(windows) == 0 && w.Manual == true) || (len(windows) > 0 && !Contains(windows, w.Name)) {
			continue
		}

		windowRoot := ExpandPath(w.Root)
		if windowRoot == "" || !filepath.IsAbs(windowRoot) {
			windowRoot = filepath.Join(sessionRoot, w.Root)
		}

		window, err := smug.tmux.NewWindow(ses, w.Name, windowRoot, w.Commands)
		if err != nil {
			return err
		}

		for _, p := range w.Panes {
			paneRoot := ExpandPath(p.Root)
			if paneRoot == "" || !filepath.IsAbs(p.Root) {
				paneRoot = filepath.Join(windowRoot, p.Root)
			}

			smug.tmux.SplitWindow(window, p.Type, paneRoot, p.Commands)
		}
	}

	if len(windows) == 0 {
		err = smug.tmux.KillWindow(ses + "0")
		if err != nil {
			return err
		}

		err = smug.tmux.RenumberWindows()
		if err != nil {
			return err
		}

		err = smug.tmux.Attach(ses+"0", os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
	}

	return nil
}
