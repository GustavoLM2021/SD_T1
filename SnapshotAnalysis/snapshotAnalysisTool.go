package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Pega o diretório de trabalho atual
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Erro ao obter o diretório atual: %v", err)
	}

	// Estrutura para armazenar as linhas dos arquivos.
	// A chave é o número da linha (int) e o valor é um slice de strings.
	linesMap := make(map[int][]string)

	// Lê o conteúdo do diretório atual
	files, err := os.ReadDir(currentDir)
	if err != nil {
		log.Fatalf("Erro ao ler o diretório: %v", err)
	}

	fmt.Println("Lendo arquivos que começam com 'snapshot'...")

	// Itera sobre todos os arquivos encontrados no diretório
	for _, file := range files {
		// Verifica se não é um diretório e se o nome começa com "snapshot"
		if !file.IsDir() && strings.HasPrefix(file.Name(), "snapshot") {
			filePath := filepath.Join(currentDir, file.Name())
			fmt.Printf("- Processando arquivo: %s\n", file.Name())

			// Abre o arquivo para leitura
			f, err := os.Open(filePath)
			if err != nil {
				log.Printf("Aviso: não foi possível abrir o arquivo %s: %v", file.Name(), err)
				continue // Pula para o próximo arquivo se houver erro
			}
			// Garante que o arquivo será fechado ao final da função
			defer f.Close()

			// Usa um scanner para ler o arquivo linha por linha
			scanner := bufio.NewScanner(f)
			lineNumber := 0
			for scanner.Scan() {
				// Adiciona a linha lida ao slice correspondente no mapa
				linesMap[lineNumber] = append(linesMap[lineNumber], scanner.Text())
				lineNumber++
			}

			// Verifica se ocorreu algum erro durante a leitura do arquivo
			if err := scanner.Err(); err != nil {
				log.Printf("Aviso: erro ao ler o arquivo %s: %v", file.Name(), err)
			}
		}
	}

	// Imprime o resultado final para verificação
	fmt.Println("\n--- Conteúdo Agrupado por Linha ---")
	for i := 0; i < len(linesMap); i++ {
		fmt.Printf("Linha %d: %v\n", i, linesMap[i])
	}
}
