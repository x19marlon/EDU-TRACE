package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"filippo.io/age"
	"github.com/marlon/edu-trace/internal/backup"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	keepDefault, _ := strconv.Atoi(env("BACKUP_KEEP", "14"))
	dataDir := flag.String("datos", env("DATA_DIR", "data"), "carpeta de datos de EduTrace")
	destDir := flag.String("destino", env("BACKUP_DIR", "respaldos"), "carpeta donde se guardan los respaldos")
	keep := flag.Int("conservar", keepDefault, "cuántos respaldos guardar (los más antiguos se borran)")
	recipient := flag.String("destinatario", os.Getenv("BACKUP_RECIPIENT"), "clave pública age (age1…) con la que se cifran los respaldos")
	identity := flag.String("identidad", os.Getenv("BACKUP_IDENTITY"), "archivo con la clave privada age para verificar o restaurar un respaldo cifrado")
	verify := flag.String("verificar", "", "solo verificar este respaldo")
	restore := flag.String("restaurar", "", "extraer este respaldo en la carpeta de -en")
	into := flag.String("en", "", "carpeta nueva donde extraer el respaldo (con -restaurar)")
	newKey := flag.Bool("generar-clave", false, "crear un par de claves age para cifrar los respaldos")
	flag.Parse()

	ids, err := backup.LoadIdentities(*identity)
	if err != nil {
		fail("no se pudo leer la clave privada %s: %v", *identity, err)
	}
	switch {
	case *newKey:
		id, err := age.GenerateX25519Identity()
		if err != nil {
			fail("no se pudo generar la clave: %v", err)
		}
		fmt.Printf("# Clave privada: guárdala FUERA del servidor (gestor de contraseñas o USB). Sin ella no se pueden restaurar los respaldos.\n%s\n\n", id)
		fmt.Printf("# Clave pública: ponla en .env como BACKUP_RECIPIENT\n%s\n", id.Recipient())
	case *verify != "":
		s, err := backup.Verify(*verify, ids)
		if err != nil {
			fail("%s no sirve: %v", *verify, err)
		}
		fmt.Printf("✓ %s está completo: %s\n", *verify, s)
	case *restore != "":
		if *into == "" {
			fail("indica con -en la carpeta nueva donde extraer el respaldo")
		}
		if err := backup.Extract(*restore, *into, ids); err != nil {
			fail("no se pudo extraer %s: %v", *restore, err)
		}
		s, err := backup.Inspect(*into)
		if err != nil {
			os.RemoveAll(*into)
			fail("la base extraída no sirve: %v", err)
		}
		fmt.Printf("✓ Respaldo extraído en %s: %s\n", *into, s)
	default:
		recipients, err := backup.ParseRecipients(*recipient)
		if err != nil {
			fail("BACKUP_RECIPIENT no es una clave pública age válida: %v", err)
		}
		start := time.Now()
		path, s, err := backup.Create(*dataDir, *destDir, start, recipients)
		if err != nil {
			fail("no se pudo crear el respaldo: %v", err)
		}
		st, _ := os.Stat(path)
		note := "sin cifrar"
		if backup.Encrypted(path) {
			note = "cifrado"
		}
		fmt.Printf("✓ %s (%.1f MB, %s, %s): %s\n", filepath.Base(path), float64(st.Size())/(1<<20), note, time.Since(start).Round(time.Millisecond), s)
		removed, err := backup.Prune(*destDir, *keep)
		if err != nil {
			fail("no se pudieron borrar los respaldos antiguos: %v", err)
		}
		for _, p := range removed {
			fmt.Printf("  borrado el respaldo antiguo %s\n", filepath.Base(p))
		}
	}
}
