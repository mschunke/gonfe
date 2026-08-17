package cte

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/internal/norm"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

// VersaoModal é a versão do grupo específico do modal.
const VersaoModal = "4.00"

// Códigos de status devolvidos pela SEFAZ.
const (
	// StatusAutorizado indica CT-e autorizado.
	StatusAutorizado = 100
	// StatusCancelado indica CT-e cancelado.
	StatusCancelado = 101
	// StatusServicoEmOperacao indica serviço em operação.
	StatusServicoEmOperacao = 107
	// StatusDenegado indica uso denegado por irregularidade.
	StatusDenegado = 110
)

// Novo devolve um CT-e com os campos de estrutura preenchidos com os padrões do
// leiaute: versão 4.00, modelo 57, emissão normal por aplicativo do
// contribuinte e o modal informado.
func Novo(modal Modal) *CTe {
	return &CTe{
		InfCte: InfCte{
			Versao: Versao,
			Ide: Ide{
				Mod:     ModeloCTe,
				Modal:   modal,
				TpEmis:  EmissaoNormal,
				ProcEmi: EmissaoAplicativoContribuinte,
				VerProc: "gonfe",
				TpCTe:   CTeNormal,
				TpImp:   Retrato,
				TpServ:  ServicoNormal,
				Retira:  RetiraNao,
			},
			InfCTeNorm: &InfCTeNorm{
				InfModal: InfModal{VersaoModal: VersaoModal},
			},
		},
	}
}

// Chave devolve a chave de acesso de 44 dígitos, extraída do atributo Id.
func (c *CTe) Chave() string { return strings.TrimPrefix(c.InfCte.Id, "CTe") }

// Modal devolve o meio de transporte da prestação.
func (c *CTe) Modal() Modal { return c.InfCte.Ide.Modal }

// Preparar deixa o conhecimento pronto para serialização: normaliza os campos
// de texto, reescala os decimais para a precisão do leiaute, calcula o valor
// total da prestação a partir dos componentes e monta a chave de acesso.
//
// Preparar é idempotente.
func (c *CTe) Preparar(opcoes ...OpcoesPreparo) error {
	opc := OpcoesPreparo{}
	if len(opcoes) > 0 {
		opc = opcoes[0]
	}

	c.InfCte.Versao = Versao
	if c.InfCte.InfCTeNorm != nil && c.InfCte.InfCTeNorm.InfModal.VersaoModal == "" {
		c.InfCte.InfCTeNorm.InfModal.VersaoModal = VersaoModal
	}
	norm.Normalizar(c)

	if !opc.SemCalculoDeTotais {
		c.CalcularTotais()
	}
	norm.Normalizar(c)

	return c.gerarChave()
}

// OpcoesPreparo ajusta o comportamento de [CTe.Preparar].
type OpcoesPreparo struct {
	// SemCalculoDeTotais preserva o grupo vPrest como preenchido pelo
	// chamador, em vez de somá-lo a partir dos componentes.
	SemCalculoDeTotais bool
}

// CalcularTotais soma os componentes do frete no valor total da prestação.
//
// O valor a receber não é tocado: ele pode diferir do total quando há retenções
// ou quando parte do frete é paga por outra parte.
func (c *CTe) CalcularTotais() {
	comps := c.InfCte.VPrest.Comp
	if len(comps) == 0 {
		return
	}
	total := tipos.NovoDecimal(0, 2)
	for _, comp := range comps {
		total = total.Somar(comp.VComp)
	}
	c.InfCte.VPrest.VTPrest = total.ComCasas(2)
	if c.InfCte.VPrest.VRec.EhZero() {
		c.InfCte.VPrest.VRec = total.ComCasas(2)
	}
}

func (c *CTe) gerarChave() error {
	ide := &c.InfCte.Ide

	documento := c.InfCte.Emit.CNPJ
	if documento == "" {
		documento = c.InfCte.Emit.CPF
	}
	if documento == "" {
		return fmt.Errorf("cte: o emitente precisa de CNPJ ou CPF para compor a chave de acesso")
	}
	if len(documento) < 14 {
		documento = strings.Repeat("0", 14-len(documento)) + documento
	}

	if ide.DhEmi.Vazia() {
		return fmt.Errorf("cte: dhEmi precisa estar preenchida para compor a chave de acesso")
	}
	ano, mes := tipos.AnoMes(ide.DhEmi.Time)

	if ide.CCT == "" {
		codigo, err := chave.GerarCodigoNumerico(ide.NCT)
		if err != nil {
			return err
		}
		ide.CCT = codigo
	}

	unidade, err := uf.PorSigla(c.InfCte.Emit.EnderEmit.UF)
	if err != nil {
		return fmt.Errorf("cte: UF do emitente: %w", err)
	}
	if ide.CUF == 0 {
		ide.CUF = unidade.Codigo()
	}

	completa, err := chave.Nova(chave.Chave{
		CUF:    ide.CUF,
		Ano:    ano,
		Mes:    mes,
		CNPJ:   documento,
		Modelo: ide.Mod.Numero(),
		Serie:  ide.Serie,
		Numero: ide.NCT,
		TpEmis: ide.TpEmis.Numero(),
		CNF:    ide.CCT,
	})
	if err != nil {
		return err
	}

	ide.CDV = int(completa[43] - '0')
	c.InfCte.Id = "CTe" + completa
	return nil
}

// XML serializa o conhecimento, sem declaração XML e sem espaços supérfluos.
func (c *CTe) XML() ([]byte, error) {
	dados, err := xml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("cte: falha ao serializar: %w", err)
	}
	return dados, nil
}

// AssinarCom prepara, serializa e assina o conhecimento em uma única chamada.
func (c *CTe) AssinarCom(assinante xmldsig.Assinante) ([]byte, error) {
	if err := c.Preparar(); err != nil {
		return nil, err
	}
	documento, err := c.XML()
	if err != nil {
		return nil, err
	}
	return xmldsig.Assinar(documento, "infCte", assinante)
}

// Ler interpreta o XML de um CT-e. Aceita o elemento <CTe> isolado ou dentro de
// um <cteProc>.
func Ler(dados []byte) (*CTe, error) {
	recorte, err := recortar(dados, "CTe")
	if err != nil {
		return nil, err
	}
	var c CTe
	if err := xml.Unmarshal(recorte, &c); err != nil {
		return nil, fmt.Errorf("cte: falha ao interpretar o XML: %w", err)
	}
	return &c, nil
}

// LerCTeProc separa o conhecimento e o protocolo de um arquivo de distribuição.
func LerCTeProc(dados []byte) (*CTe, *ProtCTe, error) {
	c, err := Ler(dados)
	if err != nil {
		return nil, nil, err
	}
	recorte, err := recortar(dados, "protCTe")
	if err != nil {
		return c, nil, nil
	}
	var prot ProtCTe
	if err := xml.Unmarshal(recorte, &prot); err != nil {
		return c, nil, fmt.Errorf("cte: falha ao interpretar o protocolo: %w", err)
	}
	return c, &prot, nil
}

// Autorizado informa se o protocolo representa uma autorização de uso.
func (p *ProtCTe) Autorizado() bool {
	return p != nil && p.InfProt.CStat == StatusAutorizado
}

// Denegado informa se o uso do documento foi denegado.
func (p *ProtCTe) Denegado() bool {
	return p != nil && p.InfProt.CStat == StatusDenegado
}

// Resumo devolve uma descrição de uma linha do protocolo.
func (p *ProtCTe) Resumo() string {
	if p == nil {
		return "sem protocolo"
	}
	if p.InfProt.NProt == "" {
		return fmt.Sprintf("%d %s", p.InfProt.CStat, p.InfProt.XMotivo)
	}
	return fmt.Sprintf("%d %s (protocolo %s)", p.InfProt.CStat, p.InfProt.XMotivo, p.InfProt.NProt)
}

// MontarCTeProc envelopa o conhecimento assinado com o protocolo de
// autorização, preservando os bytes assinados.
func MontarCTeProc(cteAssinado []byte, prot *ProtCTe) ([]byte, error) {
	if prot == nil {
		return nil, fmt.Errorf("cte: protocolo ausente")
	}
	recorte, err := recortar(cteAssinado, "CTe")
	if err != nil {
		return nil, err
	}
	if prot.Versao == "" {
		prot.Versao = Versao
	}
	protXML, err := xml.Marshal(prot)
	if err != nil {
		return nil, fmt.Errorf("cte: falha ao serializar o protocolo: %w", err)
	}

	var b bytes.Buffer
	b.WriteString(`<cteProc xmlns="` + Espaco + `" versao="` + Versao + `">`)
	b.Write(recorte)
	b.Write(protXML)
	b.WriteString(`</cteProc>`)
	return b.Bytes(), nil
}

// MontarEnvioSincrono prepara a mensagem do serviço de recepção síncrona, que
// no leiaute 4.00 substituiu o envio em lote.
//
// O conteúdo vai comprimido: o serviço recebe o CT-e assinado em gzip
// codificado em base64. A compressão não altera os bytes assinados, então a
// assinatura continua conferindo do outro lado.
func MontarEnvioSincrono(cteAssinado []byte) ([]byte, error) {
	recorte, err := recortar(cteAssinado, "CTe")
	if err != nil {
		return nil, err
	}
	var comprimido bytes.Buffer
	w := gzip.NewWriter(&comprimido)
	if _, err := w.Write(recorte); err != nil {
		return nil, fmt.Errorf("cte: falha ao comprimir: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("cte: falha ao comprimir: %w", err)
	}
	return []byte(base64.StdEncoding.EncodeToString(comprimido.Bytes())), nil
}

// ProximoIdLote devolve um identificador de lote a partir de um contador.
func ProximoIdLote(contador int64) string {
	return strconv.FormatInt(contador%1_000_000_000_000_000, 10)
}

// XMLDeclarado antepõe a declaração XML em UTF-8 ao documento.
func XMLDeclarado(documento []byte) []byte {
	const declaracao = `<?xml version="1.0" encoding="UTF-8"?>`
	if bytes.HasPrefix(bytes.TrimSpace(documento), []byte("<?xml")) {
		return documento
	}
	saida := make([]byte, 0, len(declaracao)+len(documento))
	saida = append(saida, declaracao...)
	return append(saida, documento...)
}

// recortar isola o primeiro elemento com o nome informado, preservando os bytes
// originais.
func recortar(dados []byte, nome string) ([]byte, error) {
	s := string(dados)
	abertura := -1
	for _, marcador := range []string{"<" + nome + " ", "<" + nome + ">"} {
		if i := strings.Index(s, marcador); i >= 0 && (abertura < 0 || i < abertura) {
			abertura = i
		}
	}
	if abertura < 0 {
		return nil, fmt.Errorf("cte: o documento não contém um elemento <%s>", nome)
	}
	fechamento := "</" + nome + ">"
	fim := strings.LastIndex(s, fechamento)
	if fim < abertura {
		return nil, fmt.Errorf("cte: o elemento <%s> não está fechado", nome)
	}
	return dados[abertura : fim+len(fechamento)], nil
}
