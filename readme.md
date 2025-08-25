## Implementar algoritmo de snapshot (protocolo de Chandy-Lamport)

<input type="checkbox" disabled checked/> Implementar uma thread no useDIMEX-f que envia a cada x tempo uma mensagem para o 
canal de entrada dmx.Req pedindo snapshot. É melhor que apenas um processo faça isso para facilitar

<input type="checkbox" disabled /> Criar função toString para gravar o estado local do processo, contendo suas variaveis

<input type="checkbox" disabled /> Algoritmo do snapshot:

1 - Ao receber mensagem pedindo snapshot, grava estado local.

2 - Envia mensagem pedindo snapshot para todos outros processos

3 - Grava mensagens recebidas por um processo (menos o q enviou) até ele responder com mensagem de snapshot de volta

4 - Termina ao receber todas mensagens dos outros processos (menos o que enviou o pedido inicial) e escreve o resultado em um arquivo

<input type="checkbox" disabled /> Variável nova no DIMEX-Template que informa se processo está no fazendo snapshot, se sim deve gravar todas mensagens que recebe (menos o canal referente ao processo que mandou o primeiro pedido de snapshot)

<input type="checkbox" disabled /> Fazer ferramenta de análise do arquivo gerado (item 3)

