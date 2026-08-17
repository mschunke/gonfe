package cteos

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/internal/norm"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

// Novo devolve um CT-e OS com os campos de estrutura preenchidos com os padrões
// do leiaute: versão 4.00, modelo 67, modal rodoviário, emissão normal por
// aplicativo do contribuinte e o tipo de serviço informado.
func Novo(servico TipoServico) *CTeOS {
	return &CTeOS{
		InfCte: InfCte{
			Versao: Versao,
			Ide: Ide{
				Mod:     cte.ModeloCTeOS,
				Modal:   cte.ModalRodoviario,
				TpEmis:  cte.EmissaoNormal,
				ProcEmi: cte.EmissaoAplicativoContribuinte,
				VerProc: "gonfe",
				TpCTe:   cte.CTeNormal,
				TpImp:   cte.Retrato,
				TpServ:  servico,
			},
			InfCTeNorm: &InfCTeNorm{
				InfModal: InfModal{VersaoModal: VersaoModal},
			},
		},
	}
}

// Chave devolve a chave de acesso de 44 dígitos, extraída do atributo Id.
func (c *CTeOS) Chave() string { return strings.TrimPrefix(c.InfCte.Id, "CTe") }

// Servico devolve o tipo de serviço prestado.
func (c *CTeOS) Servico() TipoServico { return c.InfCte.Ide.TpServ }

// Preparar deixa o conhecimento pronto para serialização: normaliza os campos
// de texto, reescala os decimais para a precisão do leiaute, calcula o valor
// total da prestação a partir dos componentes e monta a chave de acesso.
//
// Preparar é idempotente.
func (c *CTeOS) Preparar(opcoes ...OpcoesPreparo) error {
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

// OpcoesPreparo ajusta o comportamento de [CTeOS.Preparar].
type OpcoesPreparo struct {
	// SemCalculoDeTotais preserva o grupo vPrest como preenchido pelo
	// chamador, em vez de somá-lo a partir dos componentes.
	SemCalculoDeTotais bool
}

// CalcularTotais soma os componentes no valor total da prestação, com a mesma
// regra do modelo 57.
func (c *CTeOS) CalcularTotais() {
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

func (c *CTeOS) gerarChave() error {
	ide := &c.InfCte.Ide

	documento := c.InfCte.Emit.CNPJ
	if documento == "" {
		documento = c.InfCte.Emit.CPF
	}
	if documento == "" {
		return fmt.Errorf("cteos: o emitente precisa de CNPJ ou CPF para compor a chave de acesso")
	}
	if len(documento) < 14 {
		documento = strings.Repeat("0", 14-len(documento)) + documento
	}

	if ide.DhEmi.Vazia() {
		return fmt.Errorf("cteos: dhEmi precisa estar preenchida para compor a chave de acesso")
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
		return fmt.Errorf("cteos: UF do emitente: %w", err)
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
	// O prefixo do atributo Id é "CTe" também no modelo 67; é a chave que
	// distingue os dois, pela posição do modelo.
	c.InfCte.Id = "CTe" + completa
	return nil
}

// XML serializa o conhecimento, sem declaração XML e sem espaços supérfluos.
func (c *CTeOS) XML() ([]byte, error) {
	dados, err := xml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("cteos: falha ao serializar: %w", err)
	}
	return dados, nil
}

// AssinarCom prepara, serializa e assina o conhecimento em uma única chamada. A
// assinatura cobre o grupo infCte, como no modelo 57.
func (c *CTeOS) AssinarCom(assinante xmldsig.Assinante) ([]byte, error) {
	if err := c.Preparar(); err != nil {
		return nil, err
	}
	documento, err := c.XML()
	if err != nil {
		return nil, err
	}
	return xmldsig.Assinar(documento, "infCte", assinante)
}

// Ler interpreta o XML de um CT-e OS. Aceita o elemento <CTeOS> isolado ou
// dentro de um <cteOSProc>.
func Ler(dados []byte) (*CTeOS, error) {
	recorte, err := recortar(dados, "CTeOS")
	if err != nil {
		return nil, err
	}
	var c CTeOS
	if err := xml.Unmarshal(recorte, &c); err != nil {
		return nil, fmt.Errorf("cteos: falha ao interpretar o XML: %w", err)
	}
	return &c, nil
}

// LerCTeOSProc separa o conhecimento e o protocolo de um arquivo de
// distribuição.
func LerCTeOSProc(dados []byte) (*CTeOS, *cte.ProtCTe, error) {
	c, err := Ler(dados)
	if err != nil {
		return nil, nil, err
	}
	recorte, err := recortar(dados, "protCTe")
	if err != nil {
		return c, nil, nil
	}
	var prot cte.ProtCTe
	if err := xml.Unmarshal(recorte, &prot); err != nil {
		return c, nil, fmt.Errorf("cteos: falha ao interpretar o protocolo: %w", err)
	}
	return c, &prot, nil
}

// MontarCTeOSProc envelopa o conhecimento assinado com o protocolo de
// autorização, preservando os bytes assinados.
func MontarCTeOSProc(assinado []byte, prot *cte.ProtCTe) ([]byte, error) {
	if prot == nil {
		return nil, fmt.Errorf("cteos: protocolo ausente")
	}
	recorte, err := recortar(assinado, "CTeOS")
	if err != nil {
		return nil, err
	}
	if prot.Versao == "" {
		prot.Versao = Versao
	}
	protXML, err := xml.Marshal(prot)
	if err != nil {
		return nil, fmt.Errorf("cteos: falha ao serializar o protocolo: %w", err)
	}

	var b bytes.Buffer
	b.WriteString(`<cteOSProc xmlns="` + Espaco + `" versao="` + Versao + `">`)
	b.Write(recorte)
	b.Write(protXML)
	b.WriteString(`</cteOSProc>`)
	return b.Bytes(), nil
}

// MontarEnvioSincrono prepara a mensagem do serviço de recepção síncrona, que
// atende aos dois modelos. O conteúdo vai comprimido em gzip e codificado em
// base64, sem alterar os bytes assinados.
func MontarEnvioSincrono(assinado []byte) ([]byte, error) {
	recorte, err := recortar(assinado, "CTeOS")
	if err != nil {
		return nil, err
	}
	var comprimido bytes.Buffer
	w := gzip.NewWriter(&comprimido)
	if _, err := w.Write(recorte); err != nil {
		return nil, fmt.Errorf("cteos: falha ao comprimir: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("cteos: falha ao comprimir: %w", err)
	}
	return []byte(base64.StdEncoding.EncodeToString(comprimido.Bytes())), nil
}

// XMLDeclarado antepõe a declaração XML em UTF-8 ao documento.
func XMLDeclarado(documento []byte) []byte { return cte.XMLDeclarado(documento) }

// recortar isola o primeiro elemento com o nome informado, preservando os bytes
// originais.
//
// A busca não pode confundir <CTeOS> com <CTe>: os marcadores incluem o
// caractere que segue o nome, então "<CTe " nunca casa com "<CTeOS ".
func recortar(dados []byte, nome string) ([]byte, error) {
	s := string(dados)
	abertura := -1
	for _, marcador := range []string{"<" + nome + " ", "<" + nome + ">"} {
		if i := strings.Index(s, marcador); i >= 0 && (abertura < 0 || i < abertura) {
			abertura = i
		}
	}
	if abertura < 0 {
		return nil, fmt.Errorf("cteos: o documento não contém um elemento <%s>", nome)
	}
	fechamento := "</" + nome + ">"
	fim := strings.LastIndex(s, fechamento)
	if fim < abertura {
		return nil, fmt.Errorf("cteos: o elemento <%s> não está fechado", nome)
	}
	return dados[abertura : fim+len(fechamento)], nil
}
