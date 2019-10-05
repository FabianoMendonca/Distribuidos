package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
)

const (
	CONN_PORT = ":3333"
	CONN_TYPE = "tcp"

	MSG_DISCONNECT = "Desconectado do servidor.\n"
)

var wg sync.WaitGroup

/*
// Funcao que le o conteudo do arquivo e retorna um slice the string com todas as linhas do arquivo
func lerTexto(caminhoDoArquivo string) ([]string, error) {
	// Abre o arquivo
	arquivo, err := os.Open(caminhoDoArquivo)
	// Caso tenha encontrado algum erro ao tentar abrir o arquivo retorne o erro encontrado
	if err != nil {
		return nil, err
	}
	// Garante que o arquivo sera fechado apos o uso
	defer arquivo.Close()

	// Cria um scanner que le cada linha do arquivo
	var linhas []string
	scanner := bufio.NewScanner(arquivo)
	for scanner.Scan() {
		linhas = append(linhas, scanner.Text())
	}

	// Retorna as linhas lidas e um erro se ocorrer algum erro no scanner
	return linhas, scanner.Err()
}

// Funcao que escreve um texto no arquivo e retorna um erro caso tenha algum problema
func escreverTexto(linhas []string, caminhoDoArquivo string) error {
	// Cria o arquivo de texto
	arquivo, err := os.Create(caminhoDoArquivo)
	// Caso tenha encontrado algum erro retornar ele
	if err != nil {
		return err
	}
	// Garante que o arquivo sera fechado apos o uso
	defer arquivo.Close()

	// Cria um escritor responsavel por escrever cada linha do slice no arquivo de texto
	escritor := bufio.NewWriter(arquivo)
	for _, linha := range linhas {
		fmt.Fprintln(escritor, linha)
	}

	// Caso a funcao flush retorne um erro ele sera retornado aqui tambem
	return escritor.Flush()
}
*/
// Lê a partir do socket as saídas para o console.
func Read(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf(MSG_DISCONNECT)
			wg.Done()
			return
		}
		fmt.Print(str)
	}
}

// Lê de Stdin e envia para o soquete.
func Write(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(conn)

	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		_, err = writer.WriteString(str)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = writer.Flush()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		/*
				var conteudo []string
				conteudo, err = lerTexto("log1.txt")

				conteudo = append(conteudo, str)

				err = escreverTexto(conteudo, "log1.txt")
				if err != nil {
					log.Fatalf("Erro:", err)
				}
			}

		*/
	}
}

// Inicia um encadeamento de leitura e gravação que se conecta ao servidor através da conexão socket.
func main() {
	wg.Add(1)

	conn, err := net.Dial(CONN_TYPE, CONN_PORT)
	if err != nil {
		fmt.Println(err)
	}

	go Read(conn)
	go Write(conn)

	wg.Wait()
}
