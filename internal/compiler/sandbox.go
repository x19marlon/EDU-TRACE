package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// Directorio de trabajo del programa dentro del sandbox.
const boxDir = "/box"

// Límites del programa del estudiante (se aplican con prlimit dentro del sandbox).
const (
	runMemoryBytes  = 256 << 20 // 256 MB de espacio de direcciones
	runFileBytes    = 1 << 20   // no puede crear archivos de más de 1 MB
	runMaxOpenFiles = 64
	compileMemBytes = 1 << 30 // g++ necesita más memoria que el programa
	compileFileSize = 64 << 20
)

// CheckSandbox verifica que las herramientas necesarias existan y funcionen.
func CheckSandbox(mode string) error {
	switch mode {
	case "none":
		return nil
	case "bwrap":
	default:
		return fmt.Errorf("SANDBOX=%q no es válido (usa \"bwrap\" o \"none\")", mode)
	}
	for _, bin := range []string{"bwrap", "prlimit"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s no está instalado (en Arch: pacman -S bubblewrap util-linux; en Debian/Ubuntu: apt install bubblewrap)", bin)
		}
	}
	out, err := exec.Command("bwrap", append(baseBwrapArgs(), "--", "/usr/bin/true")...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("bwrap no puede crear el sandbox (¿user namespaces deshabilitados?): %v: %s", err, out)
	}
	return nil
}

// baseBwrapArgs monta /usr en solo lectura, sin red, sin acceso a /home ni a
// otros procesos, con un /tmp vacío y sin variables de entorno del servidor.
func baseBwrapArgs() []string {
	args := []string{
		"--ro-bind", "/usr", "/usr",
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		"--unshare-all", // red, PID, IPC, UTS, usuario y cgroup propios
		"--die-with-parent",
		"--new-session",
		"--clearenv",
		"--setenv", "PATH", "/usr/bin:/bin",
		"--setenv", "HOME", boxDir,
	}
	// /lib, /lib64, /bin y /sbin son symlinks a /usr en distros con usr-merge
	// (Arch, Debian 12+, Fedora); si son directorios reales, se montan tal cual.
	for _, p := range []string{"/lib", "/lib64", "/bin", "/sbin"} {
		fi, err := os.Lstat(p)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			if target, err := os.Readlink(p); err == nil {
				args = append(args, "--symlink", target, p)
			}
		} else if fi.IsDir() {
			args = append(args, "--ro-bind", p, p)
		}
	}
	// Algunos binarios necesitan ld.so.cache para resolver bibliotecas.
	if _, err := os.Stat("/etc/ld.so.cache"); err == nil {
		args = append(args, "--ro-bind", "/etc/ld.so.cache", "/etc/ld.so.cache")
	}
	return args
}

func prlimitArgs(mem, fsize int, cpuSecs int) []string {
	return []string{
		"prlimit",
		"--as=" + strconv.Itoa(mem),
		"--fsize=" + strconv.Itoa(fsize),
		"--nofile=" + strconv.Itoa(runMaxOpenFiles),
		"--cpu=" + strconv.Itoa(cpuSecs),
		"--core=0",
		"--",
	}
}

// compileCommand devuelve el comando (nombre + args) para compilar main.cpp.
// Se ejecuta con cmd.Dir = workDir, por eso las rutas de g++ son relativas.
func (c *Compiler) compileCommand(workDir string) (string, []string) {
	gppArgs := []string{"-o", "main", "-Wall", "-Wextra", "-std=c++17", "main.cpp"}
	if c.cfg.Sandbox == "none" {
		return c.cfg.GppPath, gppArgs
	}
	cpu := int(c.cfg.CompileTimeout.Seconds()) + 1
	args := append(baseBwrapArgs(), "--bind", workDir, boxDir, "--chdir", boxDir, "--")
	args = append(args, prlimitArgs(compileMemBytes, compileFileSize, cpu)...)
	args = append(args, c.cfg.GppPath)
	return "bwrap", append(args, gppArgs...)
}

// runCommand devuelve el comando para ejecutar el binario compilado en workDir.
func (c *Compiler) runCommand(workDir string) (string, []string) {
	if c.cfg.Sandbox == "none" {
		return filepath.Join(workDir, "main"), nil
	}
	cpu := int(c.cfg.RunTimeout.Seconds()) + 1
	// El directorio del estudiante se monta en solo lectura: el programa solo puede
	// escribir en su /tmp privado (tmpfs), que desaparece al terminar.
	args := append(baseBwrapArgs(), "--ro-bind", workDir, boxDir, "--chdir", boxDir, "--")
	args = append(args, prlimitArgs(runMemoryBytes, runFileBytes, cpu)...)
	return "bwrap", append(args, boxDir+"/main")
}
