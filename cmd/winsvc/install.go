package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func installServiceImpl() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	// If the service already exists, stop and delete it.
	if err := deleteService(m); err != nil {
		return err
	}

	// Copy ourselves somewhere.
	path, err := copyBinary()
	if err != nil {
		return err
	}

	// Install service.
	cfg := mgr.Config{
		DisplayName: "Dynatrace OneAgent Mock",
		Description: "Mock for the Dynatrace OneAgent",
		StartType:   mgr.StartAutomatic,
	}

	s, err := m.CreateService(serviceName, path, cfg, "service")
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}
	defer s.Close()

	if err = s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

func deleteService(m *mgr.Mgr) error {
	s, err := m.OpenService(serviceName)
	if err != nil { // We get an error if the service doesn't exist.
		return nil
	}
	defer s.Close()

	st, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %v", err)
	}

	if st.State == svc.Running {
		if _, err := s.Control(svc.Stop); err != nil {
			return fmt.Errorf("failed to stop service: %v", err)
		}
	}

	// Mark for deletion.
	if err = s.Delete(); err != nil {
		return fmt.Errorf("failed to delete service: %v", err)
	}

	// Wait some time to see if it gets deleted by now.
	time.Sleep(5 * time.Second)

	return nil
}

func copyBinary() (string, error) {
	const destDir = "C:\\dynatrace-mock"

	srcPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to find the path for this process: %v", err)
	}

	if os.MkdirAll(destDir, os.ModeDir) != nil {
		return "", fmt.Errorf("failed to create target directory: %v", err)
	}

	destPath := path.Join(destDir, "oneagentwatchdog.exe")

	in, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source: %v", err)
	}
	defer in.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to open target: %v", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return "", fmt.Errorf("failed to copy binary: %v", err)
	}

	return destPath, nil
}

func installService() int {
	fmt.Println("Installing service...")

	if err := installServiceImpl(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to install service:", err)
		return 1
	}

	return 0
}
