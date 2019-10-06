# SIEComp Chat

Aplicativo de bate papo (chat) distribuído desenvolvido na linguagem de programação Go Lang

Trabalho da disciplina de Sistemas Distribuídos 2019
Membros do grupo: Álvaro Spies Nolibos, Alvaro Martins, Fabiano Mendonça

A pasta CLiente possui o código dos clientes (cliente.go).
A pasta Servidor possui o código dos servidores (server.go)

O arquivo instrucoes.md da pasta Servidor possui os comandos disponíveis no chat.
O arquivo servidorconfig.json é um JSON contendo as configurações de portas iniciais para os clientes e servidores.

Para executar o código do cliente basta utilizar 'go run cliente.go' no terminal. o código irá perguntar a porta de qual servidor o cliente deseja se conectar.

Para executar o servidor deve-se informar o nome do servidor que deseja ser conectado (S1, S2, S3, S4) 
Ex:  'go run server.go S1'

# Comandos de cliente

"/create sala" criar uma sala com o nome "sala";

"/join sala" entrar na sala com o nome "sala";

"/leave" deixar a sala de chat atual;

"/list" lista todas as salas de chat;

"/name" nome muda o nome do cliente para "nome"

"/help" lista de todos os comando do chat;

"/quit" sair do programa.
