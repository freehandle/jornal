package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func CarregarOptIn(caminho string) map[string]struct{} {
	optin := make(map[string]struct{})
	f, err := os.Open(caminho)
	if err != nil {
		return optin
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			optin[line] = struct{}{}
		}
	}
	return optin
}

func SalvarOptIn(caminho string, optin map[string]struct{}) error {
	f, err := os.Create(caminho)
	if err != nil {
		return err
	}
	defer f.Close()
	for arroba := range optin {
		fmt.Fprintln(f, arroba)
	}
	return nil
}
