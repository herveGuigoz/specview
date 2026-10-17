// Build self-contained release binaries after compiling the frontend.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if err := os.MkdirAll("bin", 0755); err != nil {
		panic(err)
	}
	for _, target := range [][2]string{{"darwin", "arm64"}, {"darwin", "amd64"}, {"linux", "amd64"}, {"linux", "arm64"}, {"windows", "amd64"}, {"windows", "arm64"}} {
		name := fmt.Sprintf("specview-%s-%s", target[0], target[1])
		if target[0] == "windows" {
			name += ".exe"
		}
		cmd := exec.Command("go", "build", "-trimpath", "-o", filepath.Join("bin", name), "./cmd/specview")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+target[0], "GOARCH="+target[1])
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Println("Building", name)
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
