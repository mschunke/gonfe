// Package pdf escreve arquivos PDF simples, o suficiente para os documentos
// auxiliares dos documentos fiscais: texto, linhas, retângulos e código de
// barras.
//
// O gerador usa apenas as fontes base-14, que todo leitor de PDF já possui, e
// por isso não precisa embutir arquivo de fonte nem depender de biblioteca
// gráfica. O módulo inteiro continua sem dependências além da biblioteca
// padrão.
//
// As coordenadas são em milímetros e a origem fica no canto superior esquerdo,
// com o eixo Y crescendo para baixo — a convenção natural para diagramar um
// formulário. A conversão para o sistema do PDF, que tem origem embaixo e mede
// em pontos, acontece internamente.
package pdf

import (
	"bytes"
	"fmt"
	"strings"
)

// pontosPorMM converte milímetros em pontos tipográficos, a unidade do PDF.
const pontosPorMM = 72.0 / 25.4

// Dimensões usuais de página, em milímetros.
const (
	LarguraA4 = 210.0
	AlturaA4  = 297.0
)

// Fonte identifica uma das fontes base-14 usadas pelo gerador.
type Fonte int

const (
	// Normal é a Helvetica.
	Normal Fonte = iota
	// Negrito é a Helvetica-Bold.
	Negrito
	// Monoespacada é a Courier, útil para colunas numéricas alinhadas.
	Monoespacada
)

func (f Fonte) recurso() string {
	switch f {
	case Negrito:
		return "F2"
	case Monoespacada:
		return "F3"
	default:
		return "F1"
	}
}

func (f Fonte) nomeBase() string {
	switch f {
	case Negrito:
		return "Helvetica-Bold"
	case Monoespacada:
		return "Courier"
	default:
		return "Helvetica"
	}
}

// Estilo descreve como um texto é desenhado.
type Estilo struct {
	// Fonte a usar; o padrão é [Normal].
	Fonte Fonte
	// Tamanho em pontos; zero equivale a 6, o corpo usual do DANFE.
	Tamanho float64
	// Cinza é o tom do texto, de 0 (preto) a 1 (branco).
	Cinza float64
}

func (e Estilo) tamanho() float64 {
	if e.Tamanho <= 0 {
		return 6
	}
	return e.Tamanho
}

// PDF acumula as páginas de um documento.
type PDF struct {
	paginas []*Pagina
}

// Novo cria um documento vazio.
func Novo() *PDF { return &PDF{} }

// Pagina é uma página em construção.
type Pagina struct {
	largura, altura float64 // em milímetros
	conteudo        bytes.Buffer
}

// NovaPagina acrescenta uma página com as dimensões informadas, em milímetros.
func (p *PDF) NovaPagina(largura, altura float64) *Pagina {
	pg := &Pagina{largura: largura, altura: altura}
	p.paginas = append(p.paginas, pg)
	return pg
}

// NovaPaginaA4 acrescenta uma página A4 em retrato.
func (p *PDF) NovaPaginaA4() *Pagina { return p.NovaPagina(LarguraA4, AlturaA4) }

// Paginas devolve quantas páginas o documento tem.
func (p *PDF) Paginas() int { return len(p.paginas) }

// Largura devolve a largura da página em milímetros.
func (pg *Pagina) Largura() float64 { return pg.largura }

// Altura devolve a altura da página em milímetros.
func (pg *Pagina) Altura() float64 { return pg.altura }

// y converte a coordenada vertical do sistema de cima para baixo, em
// milímetros, para o sistema do PDF, de baixo para cima, em pontos.
func (pg *Pagina) y(mm float64) float64 { return (pg.altura - mm) * pontosPorMM }

func x(mm float64) float64 { return mm * pontosPorMM }

// Texto desenha o texto com o canto superior esquerdo em (posX, posY).
func (pg *Pagina) Texto(posX, posY float64, texto string, e Estilo) {
	if texto == "" {
		return
	}
	tamanho := e.tamanho()
	// A linha de base fica abaixo do topo do texto; 0,8 do corpo é uma
	// aproximação boa o suficiente para as fontes base-14.
	base := pg.y(posY) - tamanho*0.8

	fmt.Fprintf(&pg.conteudo, "BT\n")
	if e.Cinza > 0 {
		fmt.Fprintf(&pg.conteudo, "%.3f g\n", e.Cinza)
	}
	fmt.Fprintf(&pg.conteudo, "/%s %.2f Tf\n", e.Fonte.recurso(), tamanho)
	fmt.Fprintf(&pg.conteudo, "%.2f %.2f Td\n", x(posX), base)
	fmt.Fprintf(&pg.conteudo, "(%s) Tj\n", escapar(paraWinAnsi(texto)))
	fmt.Fprintf(&pg.conteudo, "ET\n")
	if e.Cinza > 0 {
		pg.conteudo.WriteString("0 g\n")
	}
}

// TextoCentralizado desenha o texto centralizado na faixa horizontal que começa
// em posX e tem a largura informada.
func (pg *Pagina) TextoCentralizado(posX, largura, posY float64, texto string, e Estilo) {
	sobra := largura - Medir(texto, e.Fonte, e.tamanho())
	if sobra < 0 {
		sobra = 0
	}
	pg.Texto(posX+sobra/2, posY, texto, e)
}

// TextoDireita desenha o texto alinhado à direita, terminando em posX mais a
// largura informada.
func (pg *Pagina) TextoDireita(posX, largura, posY float64, texto string, e Estilo) {
	sobra := largura - Medir(texto, e.Fonte, e.tamanho())
	if sobra < 0 {
		sobra = 0
	}
	pg.Texto(posX+sobra, posY, texto, e)
}

// Linha desenha um segmento de reta.
func (pg *Pagina) Linha(x1, y1, x2, y2, espessura float64) {
	fmt.Fprintf(&pg.conteudo, "%.2f w\n%.2f %.2f m\n%.2f %.2f l\nS\n",
		espessura*pontosPorMM, x(x1), pg.y(y1), x(x2), pg.y(y2))
}

// Retangulo desenha o contorno de um retângulo com o canto superior esquerdo em
// (posX, posY).
func (pg *Pagina) Retangulo(posX, posY, largura, altura, espessura float64) {
	fmt.Fprintf(&pg.conteudo, "%.2f w\n%.2f %.2f %.2f %.2f re\nS\n",
		espessura*pontosPorMM, x(posX), pg.y(posY+altura), x(largura), x(altura))
}

// RetanguloPreenchido desenha um retângulo cheio no tom de cinza informado, de
// 0 (preto) a 1 (branco).
func (pg *Pagina) RetanguloPreenchido(posX, posY, largura, altura, cinza float64) {
	fmt.Fprintf(&pg.conteudo, "%.3f g\n%.2f %.2f %.2f %.2f re\nf\n0 g\n",
		cinza, x(posX), pg.y(posY+altura), x(largura), x(altura))
}

// Medir devolve a largura que o texto ocupará, em milímetros.
func Medir(texto string, f Fonte, tamanho float64) float64 {
	var milesimos float64
	for _, r := range texto {
		milesimos += float64(larguraDoCaractere(r, f))
	}
	return milesimos / 1000 * tamanho / pontosPorMM
}

// Encurtar corta o texto até caber na largura informada, acrescentando
// reticências quando houver corte.
func Encurtar(texto string, f Fonte, tamanho, largura float64) string {
	if Medir(texto, f, tamanho) <= largura {
		return texto
	}
	const reticencias = "…"
	disponivel := largura - Medir(reticencias, f, tamanho)
	if disponivel <= 0 {
		return ""
	}
	runas := []rune(texto)
	for i := len(runas); i > 0; i-- {
		if Medir(string(runas[:i]), f, tamanho) <= disponivel {
			return strings.TrimRight(string(runas[:i]), " ") + reticencias
		}
	}
	return ""
}

// Quebrar divide o texto em linhas que cabem na largura informada, quebrando
// entre palavras sempre que possível.
func Quebrar(texto string, f Fonte, tamanho, largura float64) []string {
	var linhas []string
	for _, paragrafo := range strings.Split(texto, "\n") {
		palavras := strings.Fields(paragrafo)
		if len(palavras) == 0 {
			linhas = append(linhas, "")
			continue
		}
		atual := palavras[0]
		for _, palavra := range palavras[1:] {
			tentativa := atual + " " + palavra
			if Medir(tentativa, f, tamanho) <= largura {
				atual = tentativa
				continue
			}
			linhas = append(linhas, atual)
			atual = palavra
		}
		linhas = append(linhas, atual)
	}
	return linhas
}

// TextoQuebrado desenha o texto em várias linhas dentro da largura informada,
// respeitando o limite de linhas quando ele for maior que zero. Devolve a
// altura ocupada, em milímetros.
func (pg *Pagina) TextoQuebrado(posX, posY, largura float64, texto string, e Estilo, maxLinhas int) float64 {
	alturaLinha := e.tamanho() / pontosPorMM * 1.15
	linhas := Quebrar(texto, e.Fonte, e.tamanho(), largura)
	if maxLinhas > 0 && len(linhas) > maxLinhas {
		linhas = linhas[:maxLinhas]
		ultima := linhas[len(linhas)-1]
		linhas[len(linhas)-1] = Encurtar(ultima+" …", e.Fonte, e.tamanho(), largura)
	}
	for i, linha := range linhas {
		pg.Texto(posX, posY+float64(i)*alturaLinha, linha, e)
	}
	return float64(len(linhas)) * alturaLinha
}

// Bytes serializa o documento.
func (p *PDF) Bytes() ([]byte, error) {
	if len(p.paginas) == 0 {
		return nil, fmt.Errorf("pdf: documento sem páginas")
	}

	var saida bytes.Buffer
	var deslocamentos []int
	objeto := func(corpo string) {
		deslocamentos = append(deslocamentos, saida.Len())
		fmt.Fprintf(&saida, "%d 0 obj\n%s\nendobj\n", len(deslocamentos), corpo)
	}

	saida.WriteString("%PDF-1.4\n")
	// O comentário binário evita que ferramentas tratem o arquivo como texto.
	saida.WriteString("%\xE2\xE3\xCF\xD3\n")

	// Numeração: 1 catálogo, 2 árvore de páginas, 3–5 fontes, e a partir de 6
	// os pares página/conteúdo.
	const primeiraPagina = 6
	var refPaginas []string
	for i := range p.paginas {
		refPaginas = append(refPaginas, fmt.Sprintf("%d 0 R", primeiraPagina+i*2))
	}

	objeto("<< /Type /Catalog /Pages 2 0 R >>")
	objeto(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>",
		strings.Join(refPaginas, " "), len(p.paginas)))
	for _, f := range []Fonte{Normal, Negrito, Monoespacada} {
		objeto(fmt.Sprintf(
			"<< /Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding >>",
			f.nomeBase()))
	}

	for i, pg := range p.paginas {
		conteudoRef := primeiraPagina + i*2 + 1
		objeto(fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] "+
				"/Resources << /Font << /F1 3 0 R /F2 4 0 R /F3 5 0 R >> >> /Contents %d 0 R >>",
			x(pg.largura), x(pg.altura), conteudoRef))

		fluxo := pg.conteudo.String()
		objeto(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(fluxo), fluxo))
	}

	inicioXref := saida.Len()
	fmt.Fprintf(&saida, "xref\n0 %d\n0000000000 65535 f \n", len(deslocamentos)+1)
	for _, d := range deslocamentos {
		fmt.Fprintf(&saida, "%010d 00000 n \n", d)
	}
	fmt.Fprintf(&saida, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(deslocamentos)+1, inicioXref)

	return saida.Bytes(), nil
}

// escapar protege os caracteres que têm significado em uma string PDF.
func escapar(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '(', ')', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
