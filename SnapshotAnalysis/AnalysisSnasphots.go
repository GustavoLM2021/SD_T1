package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func readSnapshotFiles() (map[int][]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Erro ao obter o diretório atual: %v", err)
	}

	linesMap := make(map[int][]string)

	files, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o diretório: %v", err)
	}

	for _, file := range files {
		// Verifica se não é um diretório e se o nome começa com "snapshot"
		if !file.IsDir() && strings.HasPrefix(file.Name(), "snapshot") {
			filePath := filepath.Join(currentDir, file.Name())

			// Abre o arquivo para leitura
			f, err := os.Open(filePath)
			if err != nil {
				log.Printf("Aviso: não foi possível abrir o arquivo %s: %v", file.Name(), err)
				os.Exit(1)
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
				os.Exit(1)
			}
		}
	}
	return linesMap, nil
}

// escreva uma ferramenta que avalia para cada snapshot se os estados dos processos estão consistentes.
//     Para cada snapshot SnId a ferramenta lê os estados gravados por cada processo, respectivo ao snapshot SnId,
//     e avalia se o mesmo está correto.
//     Para isso voce tem que enunciar invariantes do sistema.   Invariante é algo que deve ser verdade em qualquer estado.
//     Exemplos
//       Inv  1:   no máximo um processo na SC.
//       inv  2:  se todos processos estão em "não quero a SC", então todos waitings tem que ser falsos e não deve haver mensagens
//       inv 3:   se um processo q está marcado como waiting em p, então p está na SC ou quer a SC
//       inv 4:   se um processo q quer a seção crítica (nao entrou ainda),
//                  então o somatório de mensagens recebidas, de mensagens em transito e de, flags waiting para p em outros processos
//                  deve ser igual a N-1  (onde N é o número total de processos)
//       inv ... etc.

//       Cada invariante é um teste sobre um snapshot, uma   funcao_InvX(snapshot)      retorna um bool com o resultado
//       Cada snapshot é avaliado para todas invariantes.
//       A ferramenta avisa invariantes violadas e o snapshot.

// func checkOnlyOneInSC(snapshot []string) bool {
// 	//TODO
// }

// func checkIfAllNotWantingThenNoWaitings(snapshot []string) bool {
// 	//TODO
// }

// func checkIfHaveWaitingThenInSCOrWanting(snapshot []string) bool {
// 	//TODO
// }

// func checkIfWantingThenMessageCount(snapshot []string) bool {
// 	//TODO
// }

func main() {
	// Pega o diretório de trabalho atual
	snapshotMap, err := readSnapshotFiles()
	if err != nil {
		log.Fatalf("Erro ao ler os arquivos de snapshot: %v", err)
	}

	// Exibe o conteúdo do mapa
	for lineNumber, lines := range snapshotMap {
		fmt.Printf("Linha %d:\n", lineNumber)
		for i, line := range lines {
			fmt.Printf("  Arquivo %d: %s\n", i+1, line)
		}
	}
}
