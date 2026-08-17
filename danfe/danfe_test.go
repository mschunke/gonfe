package danfe_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/danfe"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/nfce"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

const cnpjExemplo = "12345678000195"

func notaComItens(t *testing.T, modelo nfe.Modelo, quantos int) *nfe.NFe {
	t.Helper()
	n := nfe.Nova(modelo)

	ide := &n.InfNFe.Ide
	ide.NatOp = "VENDA DE MERCADORIA ADQUIRIDA DE TERCEIROS"
	ide.Serie = 1
	ide.NNF = 4321
	ide.CNF = "99887766"
	ide.DhEmi = tipos.DH("2026-03-04T14:20:00-03:00")
	ide.CMunFG = 4314902
	ide.TpAmb = nfe.Homologacao
	ide.IndFinal = "1"
	ide.IndPres = nfe.PresencaPresencial
	if modelo == nfe.ModeloNFCe {
		ide.TpImp = nfe.DANFENFCe
	}

	n.InfNFe.Emit = nfe.Emit{
		CNPJ:  cnpjExemplo,
		XNome: "COMERCIO DE PECAS E ACESSORIOS AUTOMOTIVOS EXEMPLO LTDA",
		XFant: "EXEMPLO AUTOPECAS",
		IE:    "0961234567",
		CRT:   nfe.RegimeNormal,
		EnderEmit: nfe.Endereco{
			XLgr: "AVENIDA IPIRANGA", Nro: "1000", XCpl: "SALA 302",
			XBairro: "PRAIA DE BELAS", CMun: 4314902, XMun: "PORTO ALEGRE",
			UF: string(uf.RS), CEP: "90160091", CPais: 1058, XPais: "BRASIL",
			Fone: "5133334444",
		},
	}

	if modelo == nfe.ModeloNFe {
		n.InfNFe.Dest = &nfe.Dest{
			CNPJ:      "11222333000181",
			XNome:     nfe.TextoObrigatorioHomologacao,
			IndIEDest: nfe.NaoContribuinte,
			Email:     "compras@cliente.com.br",
			EnderDest: &nfe.Endereco{
				XLgr: "RUA DAS FLORES", Nro: "42", XBairro: "CENTRO",
				CMun: 4314902, XMun: "PORTO ALEGRE", UF: "RS", CEP: "90010000",
				CPais: 1058, XPais: "BRASIL", Fone: "5199998888",
			},
		}
	}

	total := tipos.D("0.00")
	for i := 1; i <= quantos; i++ {
		valor := tipos.D("19.90")
		total = total.Somar(valor)
		n.InfNFe.Det = append(n.InfNFe.Det, nfe.Det{
			Prod: nfe.Prod{
				CProd:    fmt.Sprintf("PC-%04d", i),
				CEAN:     "SEM GTIN",
				XProd:    fmt.Sprintf("PASTILHA DE FREIO DIANTEIRA MODELO %d COM DESCRICAO LONGA", i),
				NCM:      "87083090",
				CFOP:     "5102",
				UCom:     "UN",
				QCom:     tipos.D("1"),
				VUnCom:   valor,
				VProd:    valor,
				CEANTrib: "SEM GTIN", UTrib: "UN", QTrib: tipos.D("1"), VUnTrib: valor,
				IndTot: nfe.CompoeTotal,
			},
			Imposto: nfe.Imposto{
				ICMS: &nfe.ICMS{ICMS00: &nfe.ICMS00{
					Orig: nfe.OrigemNacional, CST: "00", ModBC: "3",
					VBC: valor, PICMS: tipos.D("18.00"), VICMS: valor.Percentual(tipos.D("18.00"), 2),
				}},
				PIS:    &nfe.PIS{PISNT: &nfe.PISNT{CST: "07"}},
				COFINS: &nfe.COFINS{COFINSNT: &nfe.COFINSNT{CST: "07"}},
			},
		})
	}

	n.InfNFe.Transp = nfe.Transp{
		ModFrete: nfe.FreteEmitente,
		Transporta: &nfe.Transporta{
			CNPJ: "99999999000191", XNome: "TRANSPORTADORA RAPIDA LTDA",
			IE: "0987654321", XEnder: "RUA DO PORTO, 500", XMun: "PORTO ALEGRE", UF: "RS",
		},
		VeicTransp: &nfe.VeicTransp{Placa: "ABC1D23", UF: "RS", RNTC: "12345678"},
		Vol: []nfe.Vol{{
			QVol: tipos.Ptr(2), Esp: "CAIXA", Marca: "EXEMPLO",
			PesoB: tipos.Ptr(tipos.D("12.500")), PesoL: tipos.Ptr(tipos.D("11.800")),
		}},
	}
	n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
		TPag: nfe.PagamentoCartaoCredito, VPag: total,
	}}}
	n.InfNFe.InfAdic = &nfe.InfAdic{
		InfCpl: "Pedido 12345. Entrega em horario comercial. " +
			"Mercadoria sujeita a conferencia no ato do recebimento.",
	}
	return n
}

func protocoloDe(n *nfe.NFe) *nfe.ProtNFe {
	return &nfe.ProtNFe{InfProt: nfe.InfProt{
		TpAmb: nfe.Homologacao, VerAplic: "RS20260304", ChNFe: n.Chave(),
		DhRecbto: tipos.DH("2026-03-04T14:20:30-03:00"),
		NProt:    "143260000012345", CStat: nfe.StatusAutorizada,
		XMotivo: "Autorizado o uso da NF-e",
	}}
}

// prepararEAssinar deixa a nota pronta e devolve o XML de distribuição.
func procDe(t *testing.T, n *nfe.NFe) []byte {
	t.Helper()
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo})
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if n.Modelo() == nfe.ModeloNFCe {
		csc := nfce.CSC{Id: "1", Codigo: "ABCDEF12-3456-7890-ABCD-EF1234567890"}
		if err := nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: csc}); err != nil {
			t.Fatalf("PreencherSuplemento: %v", err)
		}
	}
	documento, err := n.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	assinada, err := xmldsig.Assinar(documento, "infNFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}
	proc, err := nfe.MontarNFeProc(assinada, protocoloDe(n))
	if err != nil {
		t.Fatalf("MontarNFeProc: %v", err)
	}
	return proc
}

// conferirPDF valida o esqueleto do arquivo gerado.
func conferirPDF(t *testing.T, dados []byte) {
	t.Helper()
	if !bytes.HasPrefix(dados, []byte("%PDF-")) {
		t.Fatalf("não é um PDF: começa com %q", primeirosBytes(dados, 20))
	}
	if !bytes.HasSuffix(dados, []byte("%%EOF\n")) {
		t.Error("o arquivo não termina com o marcador de fim")
	}
	if !bytes.Contains(dados, []byte("/Type /Catalog")) {
		t.Error("falta o catálogo")
	}
	if len(dados) < 1000 {
		t.Errorf("o arquivo tem só %d bytes; parece vazio", len(dados))
	}
}

func primeirosBytes(b []byte, n int) string {
	if len(b) < n {
		n = len(b)
	}
	return string(b[:n])
}

func TestDANFE(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 3)
	proc := procDe(t, n)

	dados, err := danfe.Gerar(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	conferirPDF(t, dados)

	// Uma nota curta cabe em uma folha.
	if paginas := bytes.Count(dados, []byte("/Type /Page ")); paginas != 1 {
		t.Errorf("%d páginas, queria 1", paginas)
	}
}

func TestDANFEPaginaAutomaticamente(t *testing.T) {
	curta := notaComItens(t, nfe.ModeloNFe, 3)
	longa := notaComItens(t, nfe.ModeloNFe, 120)

	pdfCurto, err := danfe.Gerar(procDe(t, curta), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	pdfLongo, err := danfe.Gerar(procDe(t, longa), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}

	paginasCurto := bytes.Count(pdfCurto, []byte("/Type /Page "))
	paginasLongo := bytes.Count(pdfLongo, []byte("/Type /Page "))
	if paginasLongo <= paginasCurto {
		t.Errorf("120 itens deveriam ocupar mais folhas: %d contra %d", paginasLongo, paginasCurto)
	}
	if paginasLongo < 3 {
		t.Errorf("%d páginas para 120 itens parece pouco", paginasLongo)
	}
}

func TestDANFEComTodosOsItens(t *testing.T) {
	// Nenhum item pode se perder na paginação: a soma dos itens desenhados em
	// todas as folhas tem de bater com a nota.
	const itens = 95
	n := notaComItens(t, nfe.ModeloNFe, itens)
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	// Cada item imprime seu código, que é único.
	for i := 1; i <= itens; i++ {
		codigo := fmt.Sprintf("PC-%04d", i)
		if !bytes.Contains(dados, []byte(codigo)) {
			t.Fatalf("o item %s não aparece no DANFE", codigo)
		}
	}
}

func TestDANFEDesenhaAChaveEmCodigoDeBarras(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 2)
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	// O código de barras vira uma sequência de retângulos preenchidos.
	if barras := bytes.Count(dados, []byte(" re\nf\n")); barras < 40 {
		t.Errorf("%d retângulos preenchidos; o Code 128 da chave tem bem mais", barras)
	}
	// A chave também aparece em texto, formatada em grupos.
	if !bytes.Contains(dados, []byte(n.Chave()[:4]+" ")) {
		t.Error("a chave formatada não aparece no documento")
	}
}

func TestDANFEComTarjaDeHomologacao(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 2)
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	if !bytes.Contains(dados, []byte("SEM VALOR FISCAL")) {
		t.Error("uma nota de homologação deveria trazer a tarja")
	}
}

func TestDANFEComTarjaDeCancelamento(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 2)
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{Cancelada: true})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	if !bytes.Contains(dados, []byte("CANCELADA")) {
		t.Error("faltou a tarja de cancelamento")
	}
}

func TestDANFESemProtocolo(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 2)
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}

	dados, err := danfe.DANFE(n, nil, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("DANFE: %v", err)
	}
	conferirPDF(t, dados)
	if !bytes.Contains(dados, []byte("DOCUMENTO SEM PROTOCOLO")) {
		t.Error("uma nota sem protocolo deveria avisar")
	}
}

func TestDANFESemCanhoto(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 2)
	proc := procDe(t, n)

	comCanhoto, err := danfe.Gerar(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	semCanhoto, err := danfe.Gerar(proc, danfe.Opcoes{SemCanhoto: true})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	if !bytes.Contains(comCanhoto, []byte("RECEBEMOS DE")) {
		t.Error("o canhoto deveria estar presente por padrão")
	}
	if bytes.Contains(semCanhoto, []byte("RECEBEMOS DE")) {
		t.Error("SemCanhoto deveria omitir o recibo de entrega")
	}
}

func TestDANFEPaisagem(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 5)
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{Orientacao: danfe.Paisagem})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	conferirPDF(t, dados)
	// Em paisagem a largura passa a ser a altura do A4, 297 mm.
	if !bytes.Contains(dados, []byte("841.89 595.28")) {
		t.Error("o MediaBox não está em paisagem")
	}
}

func TestCupomNFCe(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFCe, 4)
	n.InfNFe.Ide.IdDest = nfe.DestinoInterno
	proc := procDe(t, n)

	dados, err := danfe.Gerar(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	conferirPDF(t, dados)

	for _, texto := range []string{
		"DANFE NFC-e", "CONSUMIDOR", "VALOR A PAGAR", "Protocolo de Autoriza",
	} {
		if !bytes.Contains(dados, []byte(texto)) {
			t.Errorf("faltou %q no cupom", texto)
		}
	}
	// A bobina padrão tem 80 mm, ou 226.77 pontos.
	if !bytes.Contains(dados, []byte("226.77")) {
		t.Error("a largura da bobina não é a padrão de 80 mm")
	}
}

func TestCupomCresceComOsItens(t *testing.T) {
	curto, err := danfe.Gerar(procDe(t, notaComItens(t, nfe.ModeloNFCe, 2)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	longo, err := danfe.Gerar(procDe(t, notaComItens(t, nfe.ModeloNFCe, 30)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	// O cupom não pagina: ele cresce em altura.
	if bytes.Count(longo, []byte("/Type /Page ")) != 1 {
		t.Error("o cupom deveria sair em página única")
	}
	if len(longo) <= len(curto) {
		t.Error("um cupom com mais itens deveria ser maior")
	}
	if !bytes.Contains(longo, []byte("PC-0030")) {
		t.Error("o último item não aparece no cupom")
	}
}

func TestCupomComQRCode(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFCe, 2)
	proc := procDe(t, n)

	semMatriz, err := danfe.Gerar(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	if !bytes.Contains(semMatriz, []byte("QR Code n")) {
		t.Error("sem a matriz, o cupom deveria avisar que o QR Code não foi incluído")
	}

	// Uma matriz 21×21 com metade dos módulos escuros.
	matriz := make(danfe.MatrizQR, 21)
	for i := range matriz {
		matriz[i] = make([]bool, 21)
		for j := range matriz[i] {
			matriz[i][j] = (i+j)%2 == 0
		}
	}
	comMatriz, err := danfe.Gerar(proc, danfe.Opcoes{QRCode: matriz})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	if bytes.Contains(comMatriz, []byte("QR Code n")) {
		t.Error("com a matriz, o aviso não deveria aparecer")
	}
	if len(comMatriz) <= len(semMatriz) {
		t.Error("o QR Code desenhado deveria aumentar o arquivo")
	}
}

func TestCupomComLarguraPersonalizada(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFCe, 2)
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{LarguraBobina: 58})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	// 58 mm em pontos.
	if !bytes.Contains(dados, []byte("164.41")) {
		t.Error("a largura personalizada não foi aplicada")
	}
}

func TestModeloErrado(t *testing.T) {
	nfeCompleta := notaComItens(t, nfe.ModeloNFe, 1)
	if err := nfeCompleta.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if _, err := danfe.Cupom(nfeCompleta, nil, danfe.Opcoes{}); err == nil {
		t.Error("o cupom não deveria aceitar uma NF-e modelo 55")
	}

	cupom := notaComItens(t, nfe.ModeloNFCe, 1)
	if err := cupom.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if _, err := danfe.DANFE(cupom, nil, danfe.Opcoes{}); err == nil {
		t.Error("o DANFE não deveria aceitar uma NFC-e modelo 65")
	}

	if _, err := danfe.DANFE(nil, nil, danfe.Opcoes{}); err == nil {
		t.Error("nota nula deveria falhar")
	}
	if _, err := danfe.Cupom(nil, nil, danfe.Opcoes{}); err == nil {
		t.Error("nota nula deveria falhar")
	}
}

func TestGerarRejeitaXMLInvalido(t *testing.T) {
	if _, err := danfe.Gerar([]byte("<x/>"), danfe.Opcoes{}); err == nil {
		t.Error("XML que não é NF-e deveria falhar")
	}
}

func TestMensagemDeRodape(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 2)
	const marca = "Emitido por Sistema Exemplo v3"
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{Mensagem: marca})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	if !bytes.Contains(dados, []byte("Sistema Exemplo")) {
		t.Error("a mensagem do rodapé não apareceu")
	}
}

func TestValoresAparecemFormatados(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 100) // 100 × 19,90 = 1.990,00
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	if !bytes.Contains(dados, []byte("1.990,00")) {
		t.Error("o total deveria aparecer com separador de milhar e vírgula decimal")
	}
	if bytes.Contains(dados, []byte("1990.00")) {
		t.Error("valores não deveriam aparecer no formato interno")
	}
}

func TestTextoAcentuadoEhCodificado(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFe, 1)
	n.InfNFe.Det[0].Prod.XProd = "PARAFUSO SEXTAVADO AÇO INOX 8mm"
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	// O texto sai em WinAnsi, não em UTF-8: o "Ç" vira o byte 0xC7 sozinho.
	if bytes.Contains(dados, []byte("AÇO")) {
		t.Error("o texto deveria ter sido convertido de UTF-8 para WinAnsi")
	}
	if !bytes.Contains(dados, []byte{'A', 0xC7, 'O'}) {
		t.Error("o texto acentuado não foi codificado corretamente")
	}
}

func TestSemPanicoComNotaMinima(t *testing.T) {
	// Uma nota com o mínimo preenchido não pode derrubar o gerador.
	n := nfe.Nova(nfe.ModeloNFe)
	n.InfNFe.Ide.DhEmi = tipos.DH("2026-03-04T14:20:00-03:00")
	n.InfNFe.Emit.CNPJ = cnpjExemplo
	n.InfNFe.Emit.EnderEmit.UF = "RS"
	_ = n.Preparar()

	dados, err := danfe.DANFE(n, nil, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("DANFE: %v", err)
	}
	conferirPDF(t, dados)
}

func TestCupomSemConsumidorIdentificado(t *testing.T) {
	n := notaComItens(t, nfe.ModeloNFCe, 2)
	n.InfNFe.Dest = nil
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	if !bytes.Contains(dados, []byte("CONSUMIDOR N")) {
		t.Error("o cupom deveria indicar consumidor não identificado")
	}
}

func TestSeparadorDeMilhar(t *testing.T) {
	// Exercitado indiretamente pelo DANFE, mas vale conferir os limites.
	n := notaComItens(t, nfe.ModeloNFe, 1)
	n.InfNFe.Det[0].Prod.VUnCom = tipos.D("1234567.89")
	n.InfNFe.Det[0].Prod.VProd = tipos.D("1234567.89")
	n.InfNFe.Det[0].Prod.QCom = tipos.D("1")
	n.InfNFe.Pag.DetPag[0].VPag = tipos.D("1234567.89")

	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	if !bytes.Contains(dados, []byte("1.234.567,89")) {
		t.Error("valores grandes deveriam ter separador de milhar a cada três casas")
	}
}

func TestNomeDoPacoteNaoVazaNoConteudo(t *testing.T) {
	// Um erro comum é imprimir a representação Go de um struct por engano.
	n := notaComItens(t, nfe.ModeloNFe, 2)
	dados, err := danfe.Gerar(procDe(t, n), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("Gerar: %v", err)
	}
	for _, suspeito := range []string{"%!", "&{", "0x"} {
		if strings.Contains(string(dados), suspeito) {
			t.Errorf("o conteúdo tem %q, sinal de formatação acidental", suspeito)
		}
	}
}
