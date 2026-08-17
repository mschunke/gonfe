package pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// TestPadroesCode128 confere as invariantes da tabela de padrões. Todo padrão
// de dado tem seis elementos somando onze módulos, e o de parada tem sete
// somando treze — um erro de transcrição quebra uma das duas regras.
func TestPadroesCode128(t *testing.T) {
	for valor, padrao := range padroesCode128 {
		elementos, soma := len(padrao), 0
		for _, d := range padrao {
			if d < '1' || d > '4' {
				t.Errorf("valor %d: elemento %q fora da faixa 1–4 em %q", valor, string(d), padrao)
			}
			soma += int(d - '0')
		}
		if valor == paradaCode128 {
			if elementos != 7 || soma != 13 {
				t.Errorf("padrão de parada = %q com %d elementos somando %d; queria 7 e 13",
					padrao, elementos, soma)
			}
			continue
		}
		if elementos != 6 {
			t.Errorf("valor %d: padrão %q tem %d elementos, queria 6", valor, padrao, elementos)
		}
		if soma != 11 {
			t.Errorf("valor %d: padrão %q soma %d módulos, queria 11", valor, padrao, soma)
		}
	}
}

func TestPadroesCode128SaoDistintos(t *testing.T) {
	vistos := make(map[string]int, len(padroesCode128))
	for valor, padrao := range padroesCode128 {
		if anterior, repetido := vistos[padrao]; repetido {
			t.Errorf("padrão %q se repete nos valores %d e %d", padrao, anterior, valor)
		}
		vistos[padrao] = valor
	}
}

func TestCodigoDeBarras(t *testing.T) {
	p := Novo()
	pg := p.NovaPaginaA4()

	const chave = "43260312345678000195550010000012341876543211"
	if err := pg.CodigoDeBarras(10, 10, 100, 13, chave); err != nil {
		t.Fatalf("CodigoDeBarras: %v", err)
	}
	// A chave tem 44 dígitos, ou 22 pares; com início, controle e parada são
	// 25 símbolos, e as barras saem como retângulos preenchidos.
	if n := strings.Count(pg.conteudo.String(), " re\nf\n"); n < 40 {
		t.Errorf("%d retângulos desenhados; um Code 128 de 44 dígitos tem bem mais", n)
	}
}

func TestCodigoDeBarrasRejeitaEntradaInvalida(t *testing.T) {
	p := Novo()
	pg := p.NovaPaginaA4()

	if err := pg.CodigoDeBarras(0, 0, 10, 10, ""); err == nil {
		t.Error("código vazio deveria falhar")
	}
	if err := pg.CodigoDeBarras(0, 0, 10, 10, "123"); err == nil {
		t.Error("quantidade ímpar de dígitos deveria falhar no modo C")
	}
	if err := pg.CodigoDeBarras(0, 0, 10, 10, "12ab"); err == nil {
		t.Error("conteúdo não numérico deveria falhar")
	}
}

func TestDocumentoValido(t *testing.T) {
	p := Novo()
	pg := p.NovaPaginaA4()
	pg.Texto(10, 10, "DANFE", Estilo{Fonte: Negrito, Tamanho: 12})
	pg.Retangulo(5, 5, 200, 50, 0.2)
	pg.Linha(5, 30, 205, 30, 0.1)

	dados, err := p.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}

	if !bytes.HasPrefix(dados, []byte("%PDF-1.4")) {
		t.Error("o arquivo deveria começar com o cabeçalho PDF")
	}
	if !bytes.HasSuffix(dados, []byte("%%EOF\n")) {
		t.Error("o arquivo deveria terminar com o marcador de fim")
	}
	for _, marcador := range []string{
		"/Type /Catalog", "/Type /Pages", "/Type /Page ",
		"/BaseFont /Helvetica", "/BaseFont /Helvetica-Bold", "/BaseFont /Courier",
		"/Encoding /WinAnsiEncoding", "xref", "trailer", "startxref",
	} {
		if !bytes.Contains(dados, []byte(marcador)) {
			t.Errorf("faltou %q no arquivo", marcador)
		}
	}
}

// TestXrefApontaParaOsObjetos confere que cada deslocamento da tabela de
// referências cruzadas cai exatamente no início do objeto correspondente. Um
// xref errado produz um PDF que alguns leitores abrem e outros recusam.
func TestXrefApontaParaOsObjetos(t *testing.T) {
	p := Novo()
	p.NovaPaginaA4().Texto(10, 10, "primeira", Estilo{})
	p.NovaPagina(80, 200).Texto(5, 5, "segunda", Estilo{})

	dados, err := p.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	texto := string(dados)

	inicioXref := strings.LastIndex(texto, "xref\n0 ")
	if inicioXref < 0 {
		t.Fatal("tabela xref não encontrada")
	}
	linhas := strings.Split(texto[inicioXref:], "\n")
	// linhas[0] é "xref", linhas[1] o cabeçalho da seção, linhas[2] a entrada
	// livre do objeto zero; as seguintes são os objetos.
	for i := 3; i < len(linhas); i++ {
		linha := linhas[i]
		if !strings.HasSuffix(linha, " 00000 n ") {
			break
		}
		// O deslocamento vem com zeros à esquerda; é preciso forçar a base
		// decimal, ou "0000000015" seria lido como octal.
		deslocamento, err := strconv.Atoi(strings.TrimLeft(linha[:10], "0") + "")
		if linha[:10] == "0000000000" {
			deslocamento = 0
		} else if err != nil {
			t.Fatalf("deslocamento ilegível em %q: %v", linha, err)
		}
		numero := i - 2
		esperado := strconv.Itoa(numero) + " 0 obj"
		if deslocamento >= len(texto) || !strings.HasPrefix(texto[deslocamento:], esperado) {
			trecho := ""
			if deslocamento < len(texto) {
				trecho = texto[deslocamento:min(deslocamento+20, len(texto))]
			}
			t.Errorf("o objeto %d deveria começar em %d; ali está %q", numero, deslocamento, trecho)
		}
	}
}

func TestPaginasComTamanhosDiferentes(t *testing.T) {
	p := Novo()
	a4 := p.NovaPaginaA4()
	bobina := p.NovaPagina(80, 300)

	if a4.Largura() != LarguraA4 || a4.Altura() != AlturaA4 {
		t.Errorf("A4 = %.2f × %.2f", a4.Largura(), a4.Altura())
	}
	if bobina.Largura() != 80 {
		t.Errorf("bobina = %.2f mm de largura", bobina.Largura())
	}
	if p.Paginas() != 2 {
		t.Errorf("%d páginas", p.Paginas())
	}

	dados, err := p.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	// 210 mm e 80 mm em pontos.
	for _, medida := range []string{"595.28", "226.77"} {
		if !bytes.Contains(dados, []byte(medida)) {
			t.Errorf("faltou a medida %s no MediaBox", medida)
		}
	}
}

func TestDocumentoSemPaginas(t *testing.T) {
	if _, err := Novo().Bytes(); err == nil {
		t.Error("documento sem páginas deveria falhar")
	}
}

func TestMedir(t *testing.T) {
	// Em Helvetica de 10 pontos, o espaço mede 278 milésimos do corpo, ou
	// 2,78 pontos, que são 0,98 mm.
	if got := Medir(" ", Normal, 10); got < 0.97 || got > 0.99 {
		t.Errorf("largura do espaço = %.4f mm, queria cerca de 0,98", got)
	}
	// A Courier é monoespaçada: dez caracteres medem dez vezes um.
	um := Medir("W", Monoespacada, 10)
	dez := Medir("WWWWWWWWWW", Monoespacada, 10)
	if dez < um*9.99 || dez > um*10.01 {
		t.Errorf("Courier não está monoespaçada: %.4f vs %.4f", dez, um*10)
	}
	// Texto acentuado mede o mesmo que sua base sem acento.
	if Medir("ação", Normal, 10) != Medir("acao", Normal, 10) {
		t.Error("caracteres acentuados deveriam medir como suas bases")
	}
	// O negrito é mais largo que o normal para as maiúsculas.
	if Medir("ABC", Negrito, 10) <= Medir("ABC", Normal, 10) {
		t.Error("Helvetica-Bold deveria ser mais larga que Helvetica em maiúsculas")
	}
	if Medir("", Normal, 10) != 0 {
		t.Error("texto vazio deveria medir zero")
	}
}

func TestEncurtar(t *testing.T) {
	const texto = "COMERCIO DE PECAS E ACESSORIOS AUTOMOTIVOS LTDA"
	completo := Medir(texto, Normal, 6)

	if got := Encurtar(texto, Normal, 6, completo+1); got != texto {
		t.Errorf("texto que já cabe não deveria ser cortado: %q", got)
	}
	curto := Encurtar(texto, Normal, 6, 20)
	if curto == texto {
		t.Error("o texto deveria ter sido cortado")
	}
	if !strings.HasSuffix(curto, "…") {
		t.Errorf("o corte deveria terminar em reticências: %q", curto)
	}
	if Medir(curto, Normal, 6) > 20 {
		t.Errorf("o texto cortado ainda não cabe: %.2f mm", Medir(curto, Normal, 6))
	}
	if got := Encurtar(texto, Normal, 6, 0.1); got != "" {
		t.Errorf("largura irrisória deveria devolver vazio, obtive %q", got)
	}
}

func TestQuebrar(t *testing.T) {
	const texto = "Informacoes complementares de interesse do contribuinte que ocupam varias linhas"
	linhas := Quebrar(texto, Normal, 6, 40)
	if len(linhas) < 2 {
		t.Fatalf("o texto deveria quebrar em mais de uma linha: %v", linhas)
	}
	for _, linha := range linhas {
		if largura := Medir(linha, Normal, 6); largura > 40 {
			t.Errorf("a linha %q mede %.2f mm e não cabe em 40", linha, largura)
		}
	}
	// Nenhuma palavra pode se perder na quebra.
	if strings.Join(linhas, " ") != texto {
		t.Errorf("a quebra alterou o texto:\n%q\n%q", strings.Join(linhas, " "), texto)
	}

	// Quebras explícitas viram linhas próprias.
	comParagrafos := Quebrar("primeira\nsegunda", Normal, 6, 100)
	if len(comParagrafos) != 2 {
		t.Errorf("quebras explícitas = %v", comParagrafos)
	}
}

func TestParaWinAnsi(t *testing.T) {
	casos := map[string]string{
		"ABC":     "ABC",
		"ação":    "a\xe7\xe3o",
		"José":    "Jos\xe9",
		"R$ 1,00": "R$ 1,00",
	}
	for entrada, esperado := range casos {
		if got := paraWinAnsi(entrada); got != esperado {
			t.Errorf("paraWinAnsi(%q) = %q, queria %q", entrada, got, esperado)
		}
	}
	// Um caractere fora da codificação vira "?" em vez de quebrar o arquivo.
	if got := paraWinAnsi("日本"); got != "??" {
		t.Errorf("caracteres fora do WinAnsi = %q", got)
	}
	// As reticências ficam na faixa especial do Windows-1252.
	if got := paraWinAnsi("…"); got != "\x85" {
		t.Errorf("reticências = %q", got)
	}
}

func TestEscapar(t *testing.T) {
	casos := map[string]string{
		"simples":       "simples",
		"com (parens)":  `com \(parens\)`,
		`barra \ aqui`:  `barra \\ aqui`,
		"quebra\nlinha": `quebra\nlinha`,
	}
	for entrada, esperado := range casos {
		if got := escapar(entrada); got != esperado {
			t.Errorf("escapar(%q) = %q, queria %q", entrada, got, esperado)
		}
	}
}

func TestTextoAlinhado(t *testing.T) {
	p := Novo()
	pg := p.NovaPaginaA4()

	// O texto centralizado começa depois do início da faixa, e o alinhado à
	// direita começa ainda mais adiante.
	pg.TextoCentralizado(0, 100, 10, "meio", Estilo{Tamanho: 10})
	centralizado := ultimaPosicaoX(t, pg)

	pg.conteudo.Reset()
	pg.TextoDireita(0, 100, 10, "meio", Estilo{Tamanho: 10})
	direita := ultimaPosicaoX(t, pg)

	if centralizado <= 0 {
		t.Errorf("o texto centralizado deveria ser deslocado, x = %.2f", centralizado)
	}
	if direita <= centralizado {
		t.Errorf("alinhado à direita (%.2f) deveria vir depois do centralizado (%.2f)",
			direita, centralizado)
	}
}

func TestQRCode(t *testing.T) {
	p := Novo()
	pg := p.NovaPagina(80, 200)

	var vazia MatrizQR
	pg.QRCode(10, 10, 20, vazia)
	if pg.conteudo.Len() != 0 {
		t.Error("matriz vazia não deveria desenhar nada")
	}

	m := MatrizQR{
		{true, false, true},
		{false, true, false},
		{true, false, true},
	}
	pg.QRCode(10, 10, 21, m)
	if n := strings.Count(pg.conteudo.String(), " re\nf\n"); n != 5 {
		t.Errorf("%d módulos desenhados, queria 5", n)
	}
}

// ultimaPosicaoX extrai o deslocamento horizontal do último comando Td do
// fluxo de conteúdo.
func ultimaPosicaoX(t *testing.T, pg *Pagina) float64 {
	t.Helper()
	texto := pg.conteudo.String()
	i := strings.LastIndex(texto, " Td")
	if i < 0 {
		t.Fatal("nenhum posicionamento de texto no fluxo")
	}
	linha := texto[:i]
	if j := strings.LastIndexByte(linha, '\n'); j >= 0 {
		linha = linha[j+1:]
	}
	var px, py float64
	if _, err := fmt.Sscan(linha, &px, &py); err != nil {
		t.Fatalf("posição ilegível em %q: %v", linha, err)
	}
	return px
}
