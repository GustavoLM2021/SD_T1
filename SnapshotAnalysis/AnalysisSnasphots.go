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
	Waiting  []bool
	Lcl      int
	ReqTs    int
	NbrResps int
	Messages []string
}

type Snapshot struct {
	ID        int
	Processes []ProcessState
}


func parseSnapshotLine(line string) (ProcessState, error) {
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
	waitingStr := parts[2]
	waiting := make([]bool, len(waitingStr))
	for i, char := range waitingStr {
		waiting[i] = char == '1'
	}

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
		ID:       snapshotID, // usar snapshot id como id do processo temporariamente
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
			
			processState, err := parseSnapshotLine(line)
			if err != nil {
				log.Printf("Erro ao parsear linha do processo %d no snapshot %d: %v", processID, snapshotID, err)
				continue
			}
			
			processState.ID = processID // Corrige o ID do processo
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
	allNotWanting := true
	
	// todaos os processos estao em noMX?
	for _, process := range snapshot.Processes {
		if process.State != NoMX {
			allNotWanting = false
			break
		}
	}
	
	if !allNotWanting {
		return true, "" // invariante nao se aplica
	}
	
	// se todos estao em noMX, verifica se nao tem waitings nem mensagens
	for _, process := range snapshot.Processes {
		// olha por waitings
		for i, waiting := range process.Waiting {
			if waiting {
				return false, fmt.Sprintf("Violação: Processo %d tem waiting[%d]=true mas todos estão em noMX", process.ID, i)
			}
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
	// usar um mapa para facilitar o acesso aos processos por id
	processMap := make(map[int]ProcessState)
	for _, process := range snapshot.Processes {
		processMap[process.ID] = process
	}
	
	for _, process := range snapshot.Processes {
		for waitingProcessID, isWaiting := range process.Waiting {
			if isWaiting {
				// processo waitingProcessID esta aguardando em process.ID
				// verifica se process.ID esta na SC ou quer a SC
				if targetProcess, exists := processMap[process.ID]; exists {
					if targetProcess.State != InMX && targetProcess.State != WantMX {
						return false, fmt.Sprintf("Violação: Processo %d está waiting em processo %d, mas processo %d está em estado %v (deveria estar InMX ou WantMX)", 
							waitingProcessID, process.ID, process.ID, targetProcess.State)
					}
				}
			}
		}
	}
	
	return true, ""
}

// Invariante 4
func checkIfWantingThenMessageCount(snapshot Snapshot) (bool, string) {
	for _, process := range snapshot.Processes {
		if process.State == WantMX {
			// se um processo quer a SC, seu nbrResps deve ser <= N-1
			N := len(snapshot.Processes)
			if process.NbrResps >= N {
				return false, fmt.Sprintf("Violação: Processo %d (wantMX) tem nbrResps=%d >= N=%d", 
					process.ID, process.NbrResps, N)
			}
			
			// se nbrResps == N-1, o processo deveria estar na SC
			if process.NbrResps == N-1 && process.State != InMX {
				return false, fmt.Sprintf("Violação: Processo %d tem nbrResps=%d (N-1) mas não está na SC", 
					process.ID, process.NbrResps)
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
			
			waitingStr := ""
			for _, w := range process.Waiting {
				if w {
					waitingStr += "1"
				} else {
					waitingStr += "0"
				}
			}
			
			fmt.Printf("  Processo %d: %s, waiting=%s, lcl=%d, reqTs=%d, nbrResps=%d, msgs=%v\n", 
				process.ID, stateStr, waitingStr, process.Lcl, process.ReqTs, process.NbrResps, process.Messages)
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
