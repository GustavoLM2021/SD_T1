package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type State int

const (
	NoMX State = iota
	WantMX
	InMX
)

type ProcessState struct {
	ID       int
	State    State
	Waiting  string
	Lcl      int
	ReqTs    int
	NbrResps int
	Messages []string
}

type Snapshot struct {
	ID        int
	Processes []ProcessState
}


func parseSnapshotLine(line string, proc_id int) (ProcessState, error) {
	parts := strings.Split(line, " ")
	if len(parts) < 6 {
		return ProcessState{}, fmt.Errorf("formato inválido da linha: %s", line)
	}

	// id do snapshot (primeiro campo)
	snapshotID, err := strconv.Atoi(parts[0])
	if err != nil {
		return ProcessState{}, fmt.Errorf("erro ao parsear snapshot ID: %v", err)
	}

	// estado do processo (segundo campo)
	var state State
	switch parts[1] {
	case "noMX":
		state = NoMX
	case "wantMX":
		state = WantMX
	case "inMX":
		state = InMX
	default:
		return ProcessState{}, fmt.Errorf("estado desconhecido: %s", parts[1])
	}

	// flags de waiting (terceiro campo)
	waiting := parts[2]

	// pegar lcl, reqTs, nbrResps (quarto, quinto e sexto campos)
	lcl, err := strconv.Atoi(parts[3])
	if err != nil {
		return ProcessState{}, fmt.Errorf("erro ao parsear lcl: %v", err)
	}

	reqTs, err := strconv.Atoi(parts[4])
	if err != nil {
		return ProcessState{}, fmt.Errorf("erro ao parsear reqTs: %v", err)
	}

	nbrResps, err := strconv.Atoi(parts[5])
	if err != nil {
		return ProcessState{}, fmt.Errorf("erro ao parsear nbrResps: %v", err)
	}

	// mensagens em trânsito (a partir do sétimo campo, se existir)
	var messages []string
	if len(parts) > 6 {
		messagesPart := strings.Join(parts[6:], " ")
		if strings.Contains(messagesPart, ";;") {
			messages = strings.Split(messagesPart, ";;")
			// eemove elementos vazios
			var filteredMessages []string
			for _, msg := range messages {
				if strings.TrimSpace(msg) != "" {
					filteredMessages = append(filteredMessages, strings.TrimSpace(msg))
				}
			}
			messages = filteredMessages
		}
	}

	return ProcessState{
		ID:       proc_id, // 🔥🔥👍🌽🔥🔥🎅🏿
		State:    state,
		Waiting:  waiting,
		Lcl:      lcl,
		ReqTs:    reqTs,
		NbrResps: nbrResps,
		Messages: messages,
	}, nil
}

func readAndParseSnapshots() ([]Snapshot, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter o diretório atual: %v", err)
	}

	linesMap := make(map[int][]string)

	files, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o diretório: %v", err)
	}

	// ler todos os arquivos snapshot
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "snapshot") {
			filePath := filepath.Join(currentDir, file.Name())

			f, err := os.Open(filePath)
			if err != nil {
				log.Printf("Aviso: não foi possível abrir o arquivo %s: %v", file.Name(), err)
				continue
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			lineNumber := 0
			for scanner.Scan() {
				linesMap[lineNumber] = append(linesMap[lineNumber], scanner.Text())
				lineNumber++
			}

			if err := scanner.Err(); err != nil {
				log.Printf("Aviso: erro ao ler o arquivo %s: %v", file.Name(), err)
				continue
			}
		}
	}

	// converte para a estrutura manipulavel
	var snapshots []Snapshot
	for snapshotID, lines := range linesMap {

		var processes []ProcessState
		for processID, line := range lines {

			if strings.TrimSpace(line) == "" {
				continue
			}
			processState, err := parseSnapshotLine(line, processID)

			if err != nil {
				log.Printf("Erro ao parsear linha do processo %d no snapshot %d: %v", processID, snapshotID, err)
				continue
			}
			processes = append(processes, processState)
		}
		
		if len(processes) > 0 {
			snapshots = append(snapshots, Snapshot{
				ID:        snapshotID,
				Processes: processes,
			})
		}
	}

	return snapshots, nil
}

// invariante 1
func checkOnlyOneInSC(snapshot Snapshot) (bool, string) {
	inSCCount := 0
	var inSCProcesses []int
	
	for _, process := range snapshot.Processes {
		if process.State == InMX {
			inSCCount++
			inSCProcesses = append(inSCProcesses, process.ID)
		}
	}
	
	if inSCCount > 1 {
		return false, fmt.Sprintf("Violação: %d processos na SC simultaneamente: %v", inSCCount, inSCProcesses)
	}
	return true, ""
}

// invariante 2
func checkIfAllNotWantingThenNoWaitings(snapshot Snapshot) (bool, string) {
	// todaos os processos estao em noMX?
	for _, process := range snapshot.Processes {
		if process.State != NoMX {
			return true, "" //não se aplica caso não estejam todos em NoMX
		}
	}
	
	// todos estao em noMX, verifica se nao tem waitings nem mensagens
	for _, process := range snapshot.Processes {
		// olha por waitings
		waiting_num, _ := strconv.Atoi(process.Waiting)

		if(waiting_num > 0) {
			return false, fmt.Sprintf("Violação: Processo %d tem waiting=%s mas todos estão em noMX", process.ID, process.Waiting)
		}
		
		// olha por mensagens em transito
		if len(process.Messages) > 0 {
			return false, fmt.Sprintf("Violação: Processo %d tem mensagens em trânsito mas todos estão em noMX: %v", process.ID, process.Messages)
		}
	}
	
	return true, ""
}

// invariante 3
func checkIfHaveWaitingThenInSCOrWanting(snapshot Snapshot) (bool, string) {
	for _, process := range snapshot.Processes {
		waiting_num, _ := strconv.Atoi(process.Waiting)
		if(waiting_num > 0 && process.State == NoMX) {
			return false, fmt.Sprintf("Violação: Processo %d tem waiting=%s mas está em noMX", process.ID, process.Waiting)
		}
	}
	return true, ""
}

// Invariante 4
func checkIfWantingThenMessageCount(snapshot Snapshot) (bool, string) {
    N := len(snapshot.Processes)

    for _, process := range snapshot.Processes {
        if process.State == WantMX {
            message_count := 0

            message_count += process.NbrResps
            for _, msg := range process.Messages {
                if strings.Contains(msg, "respOk") {
                    message_count++
                }
            }

            for _, otherProcess := range snapshot.Processes {
                if otherProcess.ID != process.ID {
                    if otherProcess.Waiting[process.ID] == '1' {
                        message_count++
                    }

                    for _, msg := range otherProcess.Messages {
                        if strings.Contains(msg, fmt.Sprintf("reqEntry,%d", process.ID)) {
                            message_count++
                        }
                    }
                }
            }
            // nmr msg recebidas
            // nmr msg em transito
            // flags em outros processos
            // soma de tudo < N

            if message_count != N-1 {
                return false, fmt.Sprintf("Violação: Processo %d em wantMX e tem somatorio de mensagens (%d) >= N (%d)", process.ID, message_count, N)
            }
        }
    }

    return true, ""
}

// invariante 5
func checkTimestampConsistency(snapshot Snapshot) (bool, string) {
	for _, process := range snapshot.Processes {
		// se o processo esta WantMX ou InMX, reqTs deve ser > 0
		if (process.State == WantMX || process.State == InMX) && process.ReqTs <= 0 {
			return false, fmt.Sprintf("Violação: Processo %d em estado %v deve ter reqTs > 0, mas tem %d", 
				process.ID, process.State, process.ReqTs)
		}
		
		// reqTs deve ser <= lcl (pois foi definido quando lcl tinha esse valor)
		if process.ReqTs > process.Lcl {
			return false, fmt.Sprintf("Violação: Processo %d tem reqTs=%d > lcl=%d", 
				process.ID, process.ReqTs, process.Lcl)
		}
	}
	
	return true, ""
}

func main() {
	
	// le todos os snapshots
	snapshots, err := readAndParseSnapshots()
	if err != nil {
		log.Fatalf("Erro ao ler os snapshots: %v", err)
	}
	
	if len(snapshots) == 0 {
		fmt.Println("Nenhum snapshot encontrado.")
		return
	}
	
	fmt.Printf("Analisando %d snapshot(s)...\n\n", len(snapshots))
	
	// lista de invariantes
	invariants := []struct {
		name string
		fn   func(Snapshot) (bool, string)
	}{
		{"Invariante 1: No max. um processo na SC", checkOnlyOneInSC},
		{"Invariante 2: Se todos noMX entao sem waitings nem mensagens", checkIfAllNotWantingThenNoWaitings},
		{"Invariante 3: Se waiting[q] em p entao p esta InMX ou WantMX", checkIfHaveWaitingThenInSCOrWanting},
		{"Invariante 4: Consistencia de nbrResps para WantMX", checkIfWantingThenMessageCount},
		{"Invariante 5: Consistencia de timestamps", checkTimestampConsistency},
	}
	
	totalViolations := 0
	
	// olha cada snapshot
	for _, snapshot := range snapshots {
		fmt.Printf("--- SNAPSHOT %d ---\n", snapshot.ID)
		
		// exibe estado dos processos
		fmt.Println("Estados dos processos:")
		for _, process := range snapshot.Processes {
			stateStr := ""
			switch process.State {
			case NoMX:
				stateStr = "noMX"
			case WantMX:
				stateStr = "wantMX"
			case InMX:
				stateStr = "inMX"
			}
			
			fmt.Printf("  Processo %d: %s, waiting=%s, lcl=%d, reqTs=%d, nbrResps=%d, msgs=%v\n", 
				process.ID, stateStr, process.Waiting, process.Lcl, process.ReqTs, process.NbrResps, process.Messages)
		}
		
		fmt.Println()
		
		// testa cada invariante
		snapshotViolations := 0
		for _, invariant := range invariants {
			valid, message := invariant.fn(snapshot)
			if !valid {
				fmt.Printf("%s: %s\n", invariant.name, message)
				snapshotViolations++
				totalViolations++
			} else {
				fmt.Printf("%s: OK\n", invariant.name)
			}
		}
		
		if snapshotViolations == 0 {
			fmt.Println("Snapshot valido - todas as invariantes satisfeitas")
		} else {
			fmt.Printf("Snapshot invalido - %d problemas encontrados\n", snapshotViolations)
		}
		
		fmt.Println()
	}
	
}