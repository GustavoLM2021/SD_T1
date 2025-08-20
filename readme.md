## Implementar algoritmo de snapshot

- Implementar uma thread no useDIMEX-f que envia a cada x tempo uma mensagem para o 
canal de entrada dmx.Req pedindo snapshot. É melhor que apenas um processo faça isso para facilitar

- Algoritmo do snapshot:
1 - Ao receber mensagem pedindo snapshot, grava estado local.
2 - Envia mensagem pedindo snapshot para todos outros processos
3 - Grava mensagens recebidas por um processo até ele responder com mensagem de snapshot de volta
4 - Termina ao receber todas mensagens dos outros processos e escreve o resultado em um arquivo

- Variável nova no DIMEX-Template que informa se processo está no fazendo snapshot, se sim deve gravar todas mensagens que recebe

- Fazer ferramenta de análise do arquivo gerado (item 3)