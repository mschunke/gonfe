package mdfe

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/internal/norm"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/validacao"
	"github.com/mschunke/gonfe/xmldsig"
)

// Códigos de status devolvidos pela SEFAZ.
const (
	// StatusAutorizado indica MDF-e autorizado.
	StatusAutorizado = 100
	// StatusCancelado indica MDF-e cancelado.
	StatusCancelado = 101
	// StatusEncerrado indica MDF-e encerrado.
	StatusEncerrado = 132
	// StatusServicoEmOperacao indica serviço em operação.
	StatusServicoEmOperacao = 107
)

// Novo devolve um MDF-e com os campos de estrutura preenchidos com os padrões
// do leiaute: versão 3.00, modelo 58, emissão normal e o modal informado.
func Novo(modal Modal) *MDFe {
	return &MDFe{
		InfMDFe: InfMDFe{
			Versao: Versao,
			Ide: Ide{
				Mod:     ModeloMDFe,
				Modal:   modal,
				TpEmis:  EmissaoNormal,
				ProcEmi: EmissaoAplicativoContribuinte,
				VerProc: "gonfe",
				TpEmit:  EmitentePrestadorServico,
			},
			InfModal: InfModal{VersaoModal: VersaoModal},
			Tot:      Tot{CUnid: UnidadeKG},
		},
	}
}

// Chave devolve a chave de acesso de 44 dígitos, extraída do atributo Id.
func (m *MDFe) Chave() string { return strings.TrimPrefix(m.InfMDFe.Id, "MDFe") }

// Modal devolve o meio de transporte da viagem.
func (m *MDFe) Modal() Modal { return m.InfMDFe.Ide.Modal }

// Preparar deixa o manifesto pronto para serialização: normaliza os campos de
// texto, reescala os decimais, conta os documentos relacionados e monta a chave
// de acesso.
//
// Preparar é idempotente.
func (m *MDFe) Preparar(opcoes ...OpcoesPreparo) error {
	opc := OpcoesPreparo{}
	if len(opcoes) > 0 {
		opc = opcoes[0]
	}

	m.InfMDFe.Versao = Versao
	if m.InfMDFe.InfModal.VersaoModal == "" {
		m.InfMDFe.InfModal.VersaoModal = VersaoModal
	}
	norm.Normalizar(m)

	if !opc.SemContagemDeDocumentos {
		m.ContarDocumentos()
	}
	norm.Normalizar(m)

	return m.gerarChave()
}

// OpcoesPreparo ajusta o comportamento de [MDFe.Preparar].
type OpcoesPreparo struct {
	// SemContagemDeDocumentos preserva as quantidades do grupo tot como
	// preenchidas pelo chamador.
	SemContagemDeDocumentos bool
}

// ContarDocumentos preenche as quantidades de CT-e, NF-e e MDF-e do grupo de
// totais a partir dos documentos relacionados em cada município de
// descarregamento.
//
// O valor e o peso da carga não são calculados: eles não saem dos documentos
// relacionados, e sim da pesagem e da nota de quem embarca.
func (m *MDFe) ContarDocumentos() {
	var cte, nfe, mdfe int
	for _, mun := range m.InfMDFe.InfDoc.InfMunDescarga {
		cte += len(mun.InfCTe)
		nfe += len(mun.InfNFe)
		mdfe += len(mun.InfMDFeTransp)
	}
	m.InfMDFe.Tot.QCTe = cte
	m.InfMDFe.Tot.QNFe = nfe
	m.InfMDFe.Tot.QMDFe = mdfe
}

// Documentos devolve todas as chaves de acesso relacionadas no manifesto, em
// ordem de município de descarregamento.
func (m *MDFe) Documentos() []string {
	var chaves []string
	for _, mun := range m.InfMDFe.InfDoc.InfMunDescarga {
		for _, d := range mun.InfCTe {
			chaves = append(chaves, d.ChCTe)
		}
		for _, d := range mun.InfNFe {
			chaves = append(chaves, d.ChNFe)
		}
		for _, d := range mun.InfMDFeTransp {
			chaves = append(chaves, d.ChMDFe)
		}
	}
	return chaves
}

func (m *MDFe) gerarChave() error {
	ide := &m.InfMDFe.Ide

	documento := m.InfMDFe.Emit.CNPJ
	if documento == "" {
		documento = m.InfMDFe.Emit.CPF
	}
	if documento == "" {
		return fmt.Errorf("mdfe: o emitente precisa de CNPJ ou CPF para compor a chave de acesso")
	}
	if len(documento) < 14 {
		documento = strings.Repeat("0", 14-len(documento)) + documento
	}

	if ide.DhEmi.Vazia() {
		return fmt.Errorf("mdfe: dhEmi precisa estar preenchida para compor a chave de acesso")
	}
	ano, mes := tipos.AnoMes(ide.DhEmi.Time)

	if ide.CMDF == "" {
		codigo, err := chave.GerarCodigoNumerico(ide.NMDF)
		if err != nil {
			return err
		}
		ide.CMDF = codigo
	}

	unidade, err := uf.PorSigla(m.InfMDFe.Emit.EnderEmit.UF)
	if err != nil {
		return fmt.Errorf("mdfe: UF do emitente: %w", err)
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
		Numero: ide.NMDF,
		TpEmis: ide.TpEmis.Numero(),
		CNF:    ide.CMDF,
	})
	if err != nil {
		return err
	}

	ide.CDV = int(completa[43] - '0')
	m.InfMDFe.Id = "MDFe" + completa
	return nil
}

// XML serializa o manifesto, sem declaração XML e sem espaços supérfluos.
func (m *MDFe) XML() ([]byte, error) {
	dados, err := xml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("mdfe: falha ao serializar: %w", err)
	}
	return dados, nil
}

// AssinarCom prepara, serializa e assina o manifesto em uma única chamada.
func (m *MDFe) AssinarCom(assinante xmldsig.Assinante) ([]byte, error) {
	if err := m.Preparar(); err != nil {
		return nil, err
	}
	documento, err := m.XML()
	if err != nil {
		return nil, err
	}
	return xmldsig.Assinar(documento, "infMDFe", assinante)
}

// Ler interpreta o XML de um MDF-e. Aceita o elemento <MDFe> isolado ou dentro
// de um <mdfeProc>.
func Ler(dados []byte) (*MDFe, error) {
	recorte, err := recortar(dados, "MDFe")
	if err != nil {
		return nil, err
	}
	var m MDFe
	if err := xml.Unmarshal(recorte, &m); err != nil {
		return nil, fmt.Errorf("mdfe: falha ao interpretar o XML: %w", err)
	}
	return &m, nil
}

// LerMDFeProc separa o manifesto e o protocolo de um arquivo de distribuição.
func LerMDFeProc(dados []byte) (*MDFe, *ProtMDFe, error) {
	m, err := Ler(dados)
	if err != nil {
		return nil, nil, err
	}
	recorte, err := recortar(dados, "protMDFe")
	if err != nil {
		return m, nil, nil
	}
	var prot ProtMDFe
	if err := xml.Unmarshal(recorte, &prot); err != nil {
		return m, nil, fmt.Errorf("mdfe: falha ao interpretar o protocolo: %w", err)
	}
	return m, &prot, nil
}

// Autorizado informa se o protocolo representa uma autorização de uso.
func (p *ProtMDFe) Autorizado() bool {
	return p != nil && p.InfProt.CStat == StatusAutorizado
}

// Resumo devolve uma descrição de uma linha do protocolo.
func (p *ProtMDFe) Resumo() string {
	if p == nil {
		return "sem protocolo"
	}
	if p.InfProt.NProt == "" {
		return fmt.Sprintf("%d %s", p.InfProt.CStat, p.InfProt.XMotivo)
	}
	return fmt.Sprintf("%d %s (protocolo %s)", p.InfProt.CStat, p.InfProt.XMotivo, p.InfProt.NProt)
}

// MontarMDFeProc envelopa o manifesto assinado com o protocolo de autorização,
// preservando os bytes assinados.
func MontarMDFeProc(mdfeAssinado []byte, prot *ProtMDFe) ([]byte, error) {
	if prot == nil {
		return nil, fmt.Errorf("mdfe: protocolo ausente")
	}
	recorte, err := recortar(mdfeAssinado, "MDFe")
	if err != nil {
		return nil, err
	}
	if prot.Versao == "" {
		prot.Versao = Versao
	}
	protXML, err := xml.Marshal(prot)
	if err != nil {
		return nil, fmt.Errorf("mdfe: falha ao serializar o protocolo: %w", err)
	}

	var b bytes.Buffer
	b.WriteString(`<mdfeProc xmlns="` + Espaco + `" versao="` + Versao + `">`)
	b.Write(recorte)
	b.Write(protXML)
	b.WriteString(`</mdfeProc>`)
	return b.Bytes(), nil
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

func recortar(dados []byte, nome string) ([]byte, error) {
	s := string(dados)
	abertura := -1
	for _, marcador := range []string{"<" + nome + " ", "<" + nome + ">"} {
		if i := strings.Index(s, marcador); i >= 0 && (abertura < 0 || i < abertura) {
			abertura = i
		}
	}
	if abertura < 0 {
		return nil, fmt.Errorf("mdfe: o documento não contém um elemento <%s>", nome)
	}
	fechamento := "</" + nome + ">"
	fim := strings.LastIndex(s, fechamento)
	if fim < abertura {
		return nil, fmt.Errorf("mdfe: o elemento <%s> não está fechado", nome)
	}
	return dados[abertura : fim+len(fechamento)], nil
}

// Erro é uma inconsistência encontrada por [MDFe.Validar].
type Erro struct {
	Campo    string
	Mensagem string
}

func (e Erro) Error() string { return e.Campo + ": " + e.Mensagem }

// Erros é o conjunto de inconsistências de um manifesto.
type Erros []Erro

func (e Erros) Error() string {
	switch len(e) {
	case 0:
		return "mdfe: nenhum erro"
	case 1:
		return "mdfe: " + e[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "mdfe: %d inconsistências:", len(e))
	for _, item := range e {
		b.WriteString("\n  - ")
		b.WriteString(item.Error())
	}
	return b.String()
}

func erro(campo, formato string, args ...any) Erro {
	return Erro{Campo: campo, Mensagem: fmt.Sprintf(formato, args...)}
}

// Validar confere o manifesto contra as regras estruturais do leiaute 3.00.
func (m *MDFe) Validar() error {
	var e Erros
	ide := &m.InfMDFe.Ide

	if m.InfMDFe.Versao != Versao {
		e = append(e, erro("infMDFe@versao", "versão %q; este pacote implementa a %s", m.InfMDFe.Versao, Versao))
	}
	if ide.Mod != ModeloMDFe {
		e = append(e, erro("ide.mod", "modelo %q; o MDF-e é o 58", ide.Mod))
	}
	if ide.Serie < 0 || ide.Serie > 999 {
		e = append(e, erro("ide.serie", "série %d fora da faixa 0–999", ide.Serie))
	}
	if ide.NMDF < 1 || ide.NMDF > 999999999 {
		e = append(e, erro("ide.nMDF", "número %d fora da faixa 1–999999999", ide.NMDF))
	}
	if ide.DhEmi.Vazia() {
		e = append(e, erro("ide.dhEmi", "data e hora de emissão são obrigatórias"))
	}
	if ide.TpAmb != Producao && ide.TpAmb != Homologacao {
		e = append(e, erro("ide.tpAmb", "ambiente %q; use 1 ou 2", ide.TpAmb))
	}
	if _, err := uf.PorSigla(ide.UFIni); err != nil {
		e = append(e, erro("ide.UFIni", "%v", err))
	}
	if _, err := uf.PorSigla(ide.UFFim); err != nil {
		e = append(e, erro("ide.UFFim", "%v", err))
	}
	if len(ide.InfMunCarrega) == 0 {
		e = append(e, erro("ide.infMunCarrega", "informe ao menos um município de carregamento"))
	}
	for i, p := range ide.InfPercurso {
		if _, err := uf.PorSigla(p.UFPer); err != nil {
			e = append(e, erro(fmt.Sprintf("ide.infPercurso[%d].UFPer", i), "%v", err))
		}
	}

	// Emitente.
	emit := &m.InfMDFe.Emit
	switch {
	case emit.CNPJ != "" && emit.CPF != "":
		e = append(e, erro("emit", "informe CNPJ ou CPF, nunca os dois"))
	case emit.CNPJ != "":
		if err := validacao.ValidarCNPJ(emit.CNPJ); err != nil {
			e = append(e, erro("emit.CNPJ", "%v", err))
		}
	case emit.CPF != "":
		if err := validacao.ValidarCPF(emit.CPF); err != nil {
			e = append(e, erro("emit.CPF", "%v", err))
		}
	default:
		e = append(e, erro("emit", "o emitente precisa de CNPJ ou CPF"))
	}
	if emit.XNome == "" {
		e = append(e, erro("emit.xNome", "razão social é obrigatória"))
	}
	if emit.IE == "" {
		e = append(e, erro("emit.IE", "inscrição estadual é obrigatória"))
	}
	if _, err := uf.PorSigla(emit.EnderEmit.UF); err != nil {
		e = append(e, erro("emit.enderEmit.UF", "%v", err))
	}

	e = append(e, m.validarModal()...)
	e = append(e, m.validarDocumentos()...)
	e = append(e, m.validarTotais()...)

	if len(e) == 0 {
		return nil
	}
	return e
}

func (m *MDFe) validarModal() Erros {
	var e Erros
	im := &m.InfMDFe.InfModal

	preenchidos := map[Modal]bool{
		ModalRodoviario:  im.Rodo != nil,
		ModalAereo:       im.Aereo != nil,
		ModalAquaviario:  im.Aquav != nil,
		ModalFerroviario: im.Ferrov != nil,
	}
	quantos := 0
	for _, ok := range preenchidos {
		if ok {
			quantos++
		}
	}
	switch {
	case quantos == 0:
		e = append(e, erro("infModal", "preencha o grupo do modal %s", m.InfMDFe.Ide.Modal.Descricao()))
		return e
	case quantos > 1:
		e = append(e, erro("infModal", "preencha apenas um grupo de modal; foram %d", quantos))
	case !preenchidos[m.InfMDFe.Ide.Modal]:
		e = append(e, erro("infModal",
			"ide.modal declara %s mas o grupo preenchido é de outro modal",
			m.InfMDFe.Ide.Modal.Descricao()))
	}

	if rodo := im.Rodo; rodo != nil {
		v := &rodo.VeicTracao
		if v.Placa == "" {
			e = append(e, erro("infModal.rodo.veicTracao.placa", "a placa do veículo é obrigatória"))
		}
		if v.Tara <= 0 {
			e = append(e, erro("infModal.rodo.veicTracao.tara", "a tara precisa ser maior que zero"))
		}
		if len(v.Condutor) == 0 {
			e = append(e, erro("infModal.rodo.veicTracao.condutor", "informe ao menos um condutor"))
		}
		for i, c := range v.Condutor {
			if c.XNome == "" {
				e = append(e, erro(fmt.Sprintf("infModal.rodo.veicTracao.condutor[%d].xNome", i),
					"o nome do condutor é obrigatório"))
			}
			if err := validacao.ValidarCPF(c.CPF); err != nil {
				e = append(e, erro(fmt.Sprintf("infModal.rodo.veicTracao.condutor[%d].CPF", i), "%v", err))
			}
		}
		for i, r := range rodo.VeicReboque {
			if r.Placa == "" {
				e = append(e, erro(fmt.Sprintf("infModal.rodo.veicReboque[%d].placa", i),
					"a placa do reboque é obrigatória"))
			}
		}
	}
	return e
}

func (m *MDFe) validarDocumentos() Erros {
	var e Erros
	municipios := m.InfMDFe.InfDoc.InfMunDescarga

	if len(municipios) == 0 {
		return append(e, erro("infDoc.infMunDescarga",
			"o manifesto precisa de ao menos um município de descarregamento"))
	}

	var total int
	for i, mun := range municipios {
		campo := fmt.Sprintf("infDoc.infMunDescarga[%d]", i)
		if mun.CMunDescarga == 0 {
			e = append(e, erro(campo+".cMunDescarga", "código do município é obrigatório"))
		}
		if mun.XMunDescarga == "" {
			e = append(e, erro(campo+".xMunDescarga", "nome do município é obrigatório"))
		}

		documentos := len(mun.InfCTe) + len(mun.InfNFe) + len(mun.InfMDFeTransp)
		if documentos == 0 {
			e = append(e, erro(campo, "o município de descarregamento não tem documento relacionado"))
		}
		total += documentos

		for j, d := range mun.InfCTe {
			if err := chave.Validar(d.ChCTe); err != nil {
				e = append(e, erro(fmt.Sprintf("%s.infCTe[%d].chCTe", campo, j), "%v", err))
			}
		}
		for j, d := range mun.InfNFe {
			if err := chave.Validar(d.ChNFe); err != nil {
				e = append(e, erro(fmt.Sprintf("%s.infNFe[%d].chNFe", campo, j), "%v", err))
			}
		}
		for j, d := range mun.InfMDFeTransp {
			if err := chave.Validar(d.ChMDFe); err != nil {
				e = append(e, erro(fmt.Sprintf("%s.infMDFeTransp[%d].chMDFe", campo, j), "%v", err))
			}
		}
	}
	if total == 0 {
		e = append(e, erro("infDoc", "o manifesto precisa relacionar ao menos um documento"))
	}
	return e
}

func (m *MDFe) validarTotais() Erros {
	var e Erros
	t := &m.InfMDFe.Tot

	if t.QCarga.EhZero() || t.QCarga.Negativo() {
		e = append(e, erro("tot.qCarga", "o peso bruto da carga precisa ser maior que zero"))
	}
	if t.VCarga.Negativo() {
		e = append(e, erro("tot.vCarga", "o valor da carga não pode ser negativo"))
	}
	if t.CUnid != UnidadeKG && t.CUnid != UnidadeTON {
		e = append(e, erro("tot.cUnid", "unidade %q; use 01 (quilograma) ou 02 (tonelada)", t.CUnid))
	}

	// As quantidades declaradas precisam bater com os documentos relacionados.
	copia := *m
	copia.ContarDocumentos()
	esperado := copia.InfMDFe.Tot
	if t.QCTe != esperado.QCTe {
		e = append(e, erro("tot.qCTe", "declarados %d CT-e, mas há %d relacionados", t.QCTe, esperado.QCTe))
	}
	if t.QNFe != esperado.QNFe {
		e = append(e, erro("tot.qNFe", "declaradas %d NF-e, mas há %d relacionadas", t.QNFe, esperado.QNFe))
	}
	if t.QMDFe != esperado.QMDFe {
		e = append(e, erro("tot.qMDFe", "declarados %d MDF-e, mas há %d relacionados", t.QMDFe, esperado.QMDFe))
	}
	return e
}
