/*  Construido como parte da disciplina: FPPD - PUCRS - Escola Politecnica
    Professor: Fernando Dotti  (https://fldotti.github.io/)
    Modulo representando Algoritmo de Exclusão Mútua Distribuída:
    Semestre 2023/1
	Aspectos a observar:
	   mapeamento de módulo para estrutura
	   inicializacao
	   semantica de concorrência: cada evento é atômico
	   							  módulo trata 1 por vez
	Q U E S T A O
	   Além de obviamente entender a estrutura ...
	   Implementar o núcleo do algoritmo ja descrito, ou seja, o corpo das
	   funcoes reativas a cada entrada possível:
	   			handleUponReqEntry()  // recebe do nivel de cima (app)
				handleUponReqExit()   // recebe do nivel de cima (app)
				handleUponDeliverRespOk(msgOutro)   // recebe do nivel de baixo
				handleUponDeliverReqEntry(msgOutro) // recebe do nivel de baixo
*/

package DIMEX

import (
	PP2PLink "SD/PP2PLink"
	"fmt"
	"strconv"
	"strings"
)

// ------------------------------------------------------------------------------------
// ------- principais tipos
// ------------------------------------------------------------------------------------

type State int // enumeracao dos estados possiveis de um processo
const (
	noMX State = iota
	wantMX
	inMX
)

type dmxReq int // enumeracao dos estados possiveis de um processo
const (
	ENTER dmxReq = iota
	EXIT
	SNAPSHOT
)

type dmxResp struct { // mensagem do módulo DIMEX infrmando que pode acessar - pode ser somente um sinal (vazio)
	// mensagem para aplicacao indicando que pode prosseguir
}

type DIMEX_Module struct {
	Req       chan dmxReq  // canal para receber pedidos da aplicacao (REQ e EXIT)
	Ind       chan dmxResp // canal para informar aplicacao que pode acessar
	addresses []string     // endereco de todos, na mesma ordem
	id        int          // identificador do processo - é o indice no array de enderecos acima
	st        State        // estado deste processo na exclusao mutua distribuida
	waiting   []bool       // processos aguardando tem flag true
	lcl       int          // relogio logico local
	reqTs     int          // timestamp local da ultima requisicao deste processo
	nbrResps  int
	dbg       bool

	Pp2plink *PP2PLink.PP2PLink // acesso aa comunicacao enviar por PP2PLinq.Req  e receber por PP2PLinq.Ind

	// variaveis de controle para algoritmo de snapshot
	makingSnapshot      bool
	snapshotAnswers     int
	processState        string
	messagesInTransit   []string
	WriteSnapshotToFile bool
}

// ------------------------------------------------------------------------------------
// ------- inicializacao
// ------------------------------------------------------------------------------------

func NewDIMEX(_addresses []string, _id int, _dbg bool) *DIMEX_Module {

	p2p := PP2PLink.NewPP2PLink(_addresses[_id], _dbg)

	dmx := &DIMEX_Module{
		Req: make(chan dmxReq, 1),
		Ind: make(chan dmxResp, 1),

		addresses: _addresses,
		id:        _id,
		st:        noMX,
		waiting:   make([]bool, len(_addresses)),
		lcl:       0,
		reqTs:     0,
		dbg:       _dbg,

		Pp2plink: p2p,

		// variaveis de snapshot
		makingSnapshot:      false,
		snapshotAnswers:     0,
		messagesInTransit:   []string{},
		WriteSnapshotToFile: false,
	}

	for i := 0; i < len(dmx.waiting); i++ {
		dmx.waiting[i] = false
	}
	dmx.Start()
	dmx.outDbg("Init DIMEX!")
	return dmx
}

// ------------------------------------------------------------------------------------
// ------- nucleo do funcionamento
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) Start() {
	go func() {
		for {
			select {
			case dmxR := <-module.Req: // vindo da  aplicação
				if dmxR == ENTER {
					module.outDbg("app pede mx")
					module.handleUponReqEntry() // ENTRADA DO ALGORITMO

				} else if dmxR == EXIT {
					module.outDbg("app libera mx")
					module.handleUponReqExit() // ENTRADA DO ALGORITMO
				} else if dmxR == SNAPSHOT {
					module.outDbg("app pede snapshot")
					module.handleSnapshot(true)
				}

			case msgOutro := <-module.Pp2plink.Ind: // vindo de outro processo
				//fmt.Printf("dimex recebe da rede: ", msgOutro)
				module.outDbg("recebeu msg de outro processo: " + msgOutro.Message)
				if strings.Contains(msgOutro.Message, "respOk") {
					module.outDbg("         <<<---- recebi um OK! " + msgOutro.Message)
					module.handleUponDeliverRespOk(msgOutro) // ENTRADA DO ALGORITMO

				} else if strings.Contains(msgOutro.Message, "reqEntry") {
					module.outDbg("          <<<---- recebi uma REQ!  " + msgOutro.Message)
					module.handleUponDeliverReqEntry(msgOutro) // ENTRADA DO ALGORITMO

				} else if strings.Contains(msgOutro.Message, "msgSnapshot") {
					module.outDbg("          <<<---- recebi pedido snapshot!  " + msgOutro.Message)
					module.handleSnapshot(false)
				}
			}
		}
	}()
}

// ------------------------------------------------------------------------------------
// ------- tratamento de pedidos vindos da aplicacao
// ------- UPON ENTRY
// ------- UPON EXIT
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponReqEntry() {
	/*
					upon event [ dmx, Entry  |  r ]  do
		    			lts.ts++
		    			myTs := lts
		    			resps := 0
		    			para todo processo p
							trigger [ pl , Send | [ reqEntry, r, myTs ]
		    			estado := queroSC
	*/
	module.lcl++
	module.reqTs = module.lcl
	module.nbrResps = 0

	for i, value := range module.addresses {
		if i == module.id {
			continue // nao pode enviar a si mesmo
		}
		module.sendToLink(value, "reqEntry,"+fmt.Sprint(module.id)+","+fmt.Sprint(module.reqTs), "")
	}
	module.st = wantMX
}

func (module *DIMEX_Module) handleUponReqExit() {
	/*
						upon event [ dmx, Exit  |  r  ]  do
		       				para todo [p, r, ts ] em waiting
		          				trigger [ pl, Send | p , [ respOk, r ]  ]
		    				estado := naoQueroSC
							waiting := {}
	*/
	for i, value := range module.waiting {
		if i == module.id {
			continue // nao pode responder a si mesmo
		}
		if value {
			module.sendToLink(module.addresses[i], "respOk,"+fmt.Sprint(module.id), "")
			module.waiting[i] = false
		}
	}
	module.st = noMX
}

// ------------------------------------------------------------------------------------
// ------- tratamento de mensagens de outros processos
// ------- UPON respOK
// ------- UPON reqEntry
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponDeliverRespOk(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	/*
						upon event [ pl, Deliver | p, [ respOk, r ] ]
		      				resps++
		      				se resps = N
		    				então trigger [ dmx, Deliver | free2Access ]
		  					    estado := estouNaSC

	*/
	module.nbrResps++
	module.outDbg("Recebi OK do ID " + strings.Split(msgOutro.Message, ",")[1])
	if module.nbrResps == len(module.addresses)-1 {
		module.outDbg("resps == N, estou na SC")
		module.st = inMX
		module.Ind <- dmxResp{} // sinaliza que pode acessar o recurso
	} else {
		module.outDbg("resps < N, esperando mais")
	}
}

func (module *DIMEX_Module) handleUponDeliverReqEntry(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	// outro processo quer entrar na SC
	/*
						upon event [ pl, Deliver | p, [ reqEntry, r, rts ]  do
		     				se (estado == naoQueroSC)   OR
		        				 (estado == QueroSC AND  myTs >  ts)
							então  trigger [ pl, Send | p , [ respOk, r ]  ]
		 					senão
		        				se (estado == estouNaSC) OR
		           					 (estado == QueroSC AND  myTs < ts)
		        				então  postergados := postergados + [p, r ]
		     					lts.ts := max(lts.ts, rts.ts)
	*/

	// IMPORTANTE: Por algum motivo o msgOutro.From ta chegando com IP errado
	FromID, _ := strconv.Atoi(strings.Split(msgOutro.Message, ",")[1])
	rts, _ := strconv.Atoi(strings.Split(msgOutro.Message, ",")[2])

	if module.st == noMX || (module.st == wantMX && (module.reqTs > rts || (module.reqTs == rts && module.id > FromID))) {
		module.outDbg("responde a IP " + module.addresses[FromID] + " com respOk")
		module.sendToLink(module.addresses[FromID], "respOk,"+fmt.Sprint(module.id), "")
	} else {
		module.outDbg("nao vai conceder para ID  " + fmt.Sprint(FromID))
		module.waiting[FromID] = true // marca que esta esperando
	}
	// atualiza o timestamp local
	//module.lcl = max(module.lcl, rts)
	if rts > module.lcl {
		module.lcl = rts
	}
}

func (module *DIMEX_Module) handleSnapshot(started bool) {
	// talvez implementado

	if module.snapshotAnswers == 0 && !module.makingSnapshot {
		module.makingSnapshot = true
		module.messagesInTransit = []string{}
		if started {
			module.snapshotAnswers = 0
		} else {
			module.snapshotAnswers = 1
		}
		module.processState = module.processStateToString()
		module.messagesInTransit = []string{}
		module.outDbg("Iniciando snapshot")
		for i, value := range module.addresses {
			if i == module.id {
				continue // nao pode enviar a si mesmo
			}
			module.sendToLink(value, "msgSnapshot,"+fmt.Sprint(module.id), "")
		}
	} else {
		module.snapshotAnswers++
		if module.snapshotAnswers == len(module.addresses)-1 {
			//finaliza snapshot
			//salvar resultados
			module.WriteSnapshotToFile = true
			module.snapshotAnswers = 0
			module.makingSnapshot = false
			module.outDbg("Finalizando snapshot")
		}
	}
}

// ------------------------------------------------------------------------------------
// ------- funcoes de ajuda
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) sendToLink(address string, content string, space string) {
	module.outDbg(space + " ---->>>>   to: " + address + "     msg: " + content)
	module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
		To:      address,
		Message: content}
}

func before(oneId, oneTs, othId, othTs int) bool {
	if oneTs < othTs {
		return true
	} else if oneTs > othTs {
		return false
	} else {
		return oneId < othId
	}
}

func (module *DIMEX_Module) outDbg(s string) {
	if module.dbg {
		fmt.Println(". . . . . . . . . . . . [ DIMEX : " + s + " ]")
	}
}

func (module *DIMEX_Module) processStateToString() string {
	// nao implementado
	s := fmt.Sprint(module.id) + " " // id

	switch module.st { // estado
	case noMX:
		s += "noMX "
	case wantMX:
		s += "wantMX "
	case inMX:
		s += "inMX "
	}

	for _, v := range module.waiting { // waiting
		if v {
			s += "1"
		} else {
			s += "0"
		}
	}
	s += " "                               // separador
	s += fmt.Sprint(module.lcl) + " "      // lcl
	s += fmt.Sprint(module.reqTs) + " "    // reqTs
	s += fmt.Sprint(module.nbrResps) + " " // nbrResps
	return s
}

func (module *DIMEX_Module) SnapshotToString() string {
	s := ""
	s += module.processState
	for i, msg := range module.messagesInTransit {
		if i < len(module.messagesInTransit)-1 {
			s += msg + "---"
		} else {
			s += msg + "\n"
		}
	}
	return s
}
