// Package nfce cuida do que é específico da Nota Fiscal de Consumidor
// Eletrônica: o QR Code impresso no DANFE NFC-e e a URL de consulta pela chave
// de acesso, que juntos compõem o grupo infNFeSupl.
//
// O QR Code é gerado aqui como texto — a URL completa com os parâmetros e o
// hash. A conversão desse texto em imagem fica por conta de quem imprime o
// cupom, o que mantém a biblioteca sem dependências gráficas.
package nfce

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/uf"
)

// VersaoQRCode é a versão do QR Code implementada, conforme a Nota Técnica
// 2015/002.
const VersaoQRCode = "2"

// CSC é o Código de Segurança do Contribuinte, fornecido pela SEFAZ da unidade
// da federação onde o estabelecimento está inscrito.
//
// O código é um segredo compartilhado entre o contribuinte e o fisco: quem o
// possui consegue forjar QR Codes válidos. Guarde-o com o mesmo cuidado
// dedicado à senha do certificado digital, fora do código-fonte e fora do
// controle de versão.
type CSC struct {
	// Id é o identificador do CSC, de um a seis dígitos. É publicado no QR
	// Code, preenchido com zeros à esquerda.
	Id string
	// Codigo é o código secreto propriamente dito.
	Codigo string
}

// Valido confere se os dois campos foram preenchidos e se o identificador é
// numérico.
func (c CSC) Valido() error {
	if c.Id == "" {
		return fmt.Errorf("nfce: identificador do CSC não informado")
	}
	if len(c.Id) > 6 || !numerico(c.Id) {
		return fmt.Errorf("nfce: identificador do CSC %q deve ter até 6 dígitos numéricos", c.Id)
	}
	if c.Codigo == "" {
		return fmt.Errorf("nfce: código do CSC não informado")
	}
	return nil
}

func (c CSC) idFormatado() string { return fmt.Sprintf("%06s", c.Id) }

// Opcoes reúne o que é preciso para montar o QR Code de uma NFC-e.
type Opcoes struct {
	// CSC é o código de segurança do contribuinte.
	CSC CSC
	// URLQRCode sobrepõe o endereço da tabela interna. Informe quando a SEFAZ
	// da sua UF publicar um endereço diferente do que a biblioteca conhece.
	URLQRCode string
	// URLConsulta sobrepõe o endereço de consulta por chave de acesso.
	URLConsulta string
}

// PreencherSuplemento calcula o QR Code e a URL de consulta da nota e preenche
// o grupo infNFeSupl.
//
// A nota precisa estar preparada — com a chave de acesso montada por
// [github.com/mschunke/gonfe/nfe.NFe.Preparar] — e ainda não assinada, porque o
// infNFeSupl entra no documento antes da assinatura.
func PreencherSuplemento(n *nfe.NFe, opc Opcoes) error {
	if n.Modelo() != nfe.ModeloNFCe {
		return fmt.Errorf("nfce: o grupo infNFeSupl só existe na NFC-e; esta nota é modelo %s", n.Modelo())
	}
	chaveAcesso := n.Chave()
	if err := chave.Validar(chaveAcesso); err != nil {
		return fmt.Errorf("nfce: chave de acesso ausente ou inválida; chame Preparar antes: %w", err)
	}
	if n.InfNFe.Ide.TpEmis == nfe.EmissaoOffline {
		return fmt.Errorf("nfce: o QR Code da contingência offline usa outro conjunto de parâmetros; " +
			"monte-o com MontarQRCode conforme a Nota Técnica vigente")
	}

	unidade, err := uf.PorSigla(n.InfNFe.Emit.EnderEmit.UF)
	if err != nil {
		return fmt.Errorf("nfce: UF do emitente: %w", err)
	}
	ambiente := n.InfNFe.Ide.TpAmb

	urlQR := opc.URLQRCode
	if urlQR == "" {
		urlQR, err = URLQRCode(unidade, ambiente)
		if err != nil {
			return err
		}
	}
	urlConsulta := opc.URLConsulta
	if urlConsulta == "" {
		urlConsulta, err = URLConsulta(unidade, ambiente)
		if err != nil {
			return err
		}
	}

	qr, err := QRCodeOnline(chaveAcesso, ambiente, opc.CSC, urlQR)
	if err != nil {
		return err
	}
	n.InfNFeSupl = &nfe.InfNFeSupl{QrCode: qr, UrlChave: urlConsulta}
	return nil
}

// QRCodeOnline monta o QR Code da emissão normal, em que o parâmetro p reúne a
// chave de acesso, a versão do QR Code, o ambiente e o identificador do CSC,
// seguidos do hash que autentica o conjunto.
func QRCodeOnline(chaveAcesso string, ambiente nfe.Ambiente, csc CSC, urlBase string) (string, error) {
	if err := chave.Validar(chaveAcesso); err != nil {
		return "", fmt.Errorf("nfce: %w", err)
	}
	if err := csc.Valido(); err != nil {
		return "", err
	}
	if urlBase == "" {
		return "", fmt.Errorf("nfce: endereço do QR Code não informado")
	}
	parametros := []string{
		chave.Limpar(chaveAcesso),
		VersaoQRCode,
		string(ambiente),
		csc.idFormatado(),
	}
	return MontarQRCode(urlBase, parametros, csc.Codigo), nil
}

// MontarQRCode monta a URL do QR Code a partir dos parâmetros na ordem exigida
// pela Nota Técnica, acrescentando o hash de autenticação.
//
// Os parâmetros são unidos por barra vertical; o hash é o SHA-1, em hexadecimal
// minúsculo, da concatenação desses parâmetros com o código do CSC — que não
// aparece na URL. Use esta função diretamente quando precisar de um conjunto de
// parâmetros que a biblioteca não monta, como o da contingência offline.
func MontarQRCode(urlBase string, parametros []string, codigoCSC string) string {
	dados := strings.Join(parametros, "|")
	resumo := sha1.Sum([]byte(dados + codigoCSC))
	separador := "?"
	if strings.Contains(urlBase, "?") {
		separador = "&"
	}
	return urlBase + separador + "p=" + dados + "|" + hex.EncodeToString(resumo[:])
}

// ConferirQRCode recalcula o hash de um QR Code e confere com o informado,
// devolvendo erro quando não batem. Serve para validar cupons recebidos e para
// testar a configuração do CSC antes de emitir.
func ConferirQRCode(qrCode, codigoCSC string) error {
	_, depois, achou := strings.Cut(qrCode, "?p=")
	if !achou {
		if _, depois, achou = strings.Cut(qrCode, "&p="); !achou {
			return fmt.Errorf("nfce: o QR Code não tem o parâmetro p")
		}
	}
	corte := strings.LastIndex(depois, "|")
	if corte < 0 {
		return fmt.Errorf("nfce: o parâmetro p não tem o campo de hash")
	}
	dados, informado := depois[:corte], depois[corte+1:]

	resumo := sha1.Sum([]byte(dados + codigoCSC))
	esperado := hex.EncodeToString(resumo[:])
	if !strings.EqualFold(informado, esperado) {
		return fmt.Errorf("nfce: hash do QR Code não confere; informado %s, calculado %s", informado, esperado)
	}
	return nil
}

func numerico(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}
