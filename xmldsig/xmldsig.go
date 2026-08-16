// Package xmldsig assina e verifica documentos fiscais eletrônicos conforme o
// perfil de XML Signature adotado pela Receita Federal.
//
// O perfil é estreito e fixo, e é isso que o torna interoperável: assinatura
// envelopada (enveloped), uma única referência apontando para o atributo Id do
// elemento assinado, Canonical XML 1.0 sem comentários, resumo SHA-1,
// assinatura RSA com PKCS#1 v1.5 e o certificado do signatário embutido em
// X509Data. O elemento Signature é sempre o último filho do elemento que
// contém o bloco assinado.
//
// Para a NF-e e a NFC-e, o elemento assinado é o infNFe e o Signature entra
// como último filho do NFe. Eventos e pedidos de inutilização seguem a mesma
// mecânica, mudando apenas o nome da tag.
package xmldsig

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mschunke/gonfe/internal/xmldom"
)

// Identificadores dos algoritmos que compõem o perfil de assinatura da SEFAZ.
const (
	AlgoritmoC14N       = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"
	AlgoritmoEnvelopado = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
	AlgoritmoResumo     = "http://www.w3.org/2000/09/xmldsig#sha1"
	AlgoritmoAssinatura = "http://www.w3.org/2000/09/xmldsig#rsa-sha1"

	// EspacoAssinatura é o namespace do XML Signature.
	EspacoAssinatura = xmldom.EspacoAssinatura
)

// Erros devolvidos pelo pacote.
var (
	// ErrElementoNaoEncontrado indica que a tag a assinar não existe no
	// documento.
	ErrElementoNaoEncontrado = errors.New("xmldsig: elemento a assinar não encontrado")
	// ErrSemAtributoId indica elemento sem o atributo Id exigido pela
	// referência da assinatura.
	ErrSemAtributoId = errors.New("xmldsig: o elemento a assinar não tem atributo Id")
	// ErrJaAssinado indica que já existe um elemento Signature no lugar onde a
	// nova assinatura seria inserida.
	ErrJaAssinado = errors.New("xmldsig: o documento já está assinado")
	// ErrSemAssinatura indica documento sem elemento Signature.
	ErrSemAssinatura = errors.New("xmldsig: o documento não está assinado")
	// ErrAssinaturaInvalida indica resumo ou assinatura que não conferem.
	ErrAssinaturaInvalida = errors.New("xmldsig: assinatura inválida")
	// ErrAlgoritmoNaoSuportado indica uso de algoritmo fora do perfil da SEFAZ.
	ErrAlgoritmoNaoSuportado = errors.New("xmldsig: algoritmo fora do perfil da SEFAZ")
)

// Assinante é a fonte da assinatura. É implementado por
// [github.com/mschunke/gonfe/certificado.Certificado] e pode ser implementado
// por adaptadores de A3, HSM ou serviços remotos de assinatura.
type Assinante interface {
	// Sign produz a assinatura PKCS#1 v1.5 do resumo informado.
	Sign(aleatorio io.Reader, resumo []byte, opcoes crypto.SignerOpts) ([]byte, error)
	// DER devolve o certificado X.509 do signatário, codificado em DER.
	DER() []byte
}

// Assinar insere a assinatura do primeiro elemento chamado tag encontrado no
// documento, devolvendo o XML assinado.
//
// Os bytes originais são preservados na íntegra: a assinatura é inserida
// imediatamente antes da tag de fechamento do elemento pai, sem reserializar o
// restante do documento. Isso garante que o resumo calculado aqui continue
// válido para quem receber o XML.
func Assinar(documento []byte, tag string, assinante Assinante) ([]byte, error) {
	return assinar(documento, tag, assinante, false)
}

// AssinarTodos assina todos os elementos chamados tag encontrados no documento.
// É o caso de um lote enviNFe montado com várias NF-e ainda não assinadas.
func AssinarTodos(documento []byte, tag string, assinante Assinante) ([]byte, error) {
	return assinar(documento, tag, assinante, true)
}

func assinar(documento []byte, tag string, assinante Assinante, todos bool) ([]byte, error) {
	doc, err := xmldom.Ler(documento)
	if err != nil {
		return nil, err
	}

	alvos := doc.Raiz.BuscarTodos(tag, "")
	if len(alvos) == 0 {
		return nil, fmt.Errorf("%w: <%s>", ErrElementoNaoEncontrado, tag)
	}
	if !todos {
		alvos = alvos[:1]
	}

	type insercao struct {
		posicao int64
		xml     string
	}
	insercoes := make([]insercao, 0, len(alvos))

	for _, alvo := range alvos {
		id, ok := alvo.Atributo("Id")
		if !ok || id == "" {
			return nil, fmt.Errorf("%w: <%s>", ErrSemAtributoId, tag)
		}
		pai := alvo.Pai
		if pai == nil {
			return nil, fmt.Errorf("xmldsig: <%s> é o elemento raiz; a assinatura precisa de um elemento que a contenha", tag)
		}
		if pai.InicioTagFinal < 0 {
			return nil, fmt.Errorf("xmldsig: <%s> está escrito na forma abreviada e não comporta a assinatura", pai.NomeQualificado())
		}
		for _, irmao := range pai.Filhos {
			if irmao.Tipo == xmldom.TipoElemento && irmao.Local == "Signature" && irmao.URI == EspacoAssinatura {
				return nil, fmt.Errorf("%w: <%s> já contém um Signature", ErrJaAssinado, pai.NomeQualificado())
			}
		}

		assinatura, err := montarAssinatura(alvo, id, assinante)
		if err != nil {
			return nil, err
		}
		insercoes = append(insercoes, insercao{posicao: pai.InicioTagFinal, xml: assinatura})
	}

	// As inserções são aplicadas de trás para frente para que os
	// deslocamentos calculados sobre o documento original continuem válidos.
	saida := documento
	for i := len(insercoes) - 1; i >= 0; i-- {
		ins := insercoes[i]
		novo := make([]byte, 0, len(saida)+len(ins.xml))
		novo = append(novo, saida[:ins.posicao]...)
		novo = append(novo, ins.xml...)
		novo = append(novo, saida[ins.posicao:]...)
		saida = novo
	}
	return saida, nil
}

func montarAssinatura(alvo *xmldom.No, id string, assinante Assinante) (string, error) {
	canonico := xmldom.Canonicalizar(alvo, xmldom.OpcoesC14N{RemoverAssinaturas: true})
	resumo := sha1.Sum(canonico)
	resumoB64 := base64.StdEncoding.EncodeToString(resumo[:])

	signedInfo := montarSignedInfo(id, resumoB64, true)
	resumoSignedInfo := sha1.Sum([]byte(signedInfo))

	bruta, err := assinante.Sign(rand.Reader, resumoSignedInfo[:], crypto.SHA1)
	if err != nil {
		return "", fmt.Errorf("xmldsig: falha ao assinar: %w", err)
	}

	cert := assinante.DER()
	if len(cert) == 0 {
		return "", errors.New("xmldsig: o assinante não forneceu o certificado X.509")
	}

	var b strings.Builder
	b.WriteString(`<Signature xmlns="` + EspacoAssinatura + `">`)
	b.WriteString(montarSignedInfo(id, resumoB64, false))
	b.WriteString(`<SignatureValue>`)
	b.WriteString(base64.StdEncoding.EncodeToString(bruta))
	b.WriteString(`</SignatureValue>`)
	b.WriteString(`<KeyInfo><X509Data><X509Certificate>`)
	b.WriteString(base64.StdEncoding.EncodeToString(cert))
	b.WriteString(`</X509Certificate></X509Data></KeyInfo>`)
	b.WriteString(`</Signature>`)
	return b.String(), nil
}

// montarSignedInfo escreve o bloco SignedInfo já na forma canônica C14N 1.0.
// Com comNamespace, o namespace do XML Signature é declarado explicitamente —
// é essa a forma que entra no cálculo da assinatura, porque canonicalizar o
// SignedInfo isoladamente faz o ápice reproduzir o namespace herdado do
// Signature. Sem o parâmetro, sai a forma que é embutida no documento, onde o
// namespace já vem do elemento pai.
func montarSignedInfo(id, resumoB64 string, comNamespace bool) string {
	var b strings.Builder
	b.WriteString("<SignedInfo")
	if comNamespace {
		b.WriteString(` xmlns="` + EspacoAssinatura + `"`)
	}
	b.WriteString(">")
	b.WriteString(`<CanonicalizationMethod Algorithm="` + AlgoritmoC14N + `"></CanonicalizationMethod>`)
	b.WriteString(`<SignatureMethod Algorithm="` + AlgoritmoAssinatura + `"></SignatureMethod>`)
	b.WriteString(`<Reference URI="#` + escaparAtributo(id) + `">`)
	b.WriteString(`<Transforms>`)
	b.WriteString(`<Transform Algorithm="` + AlgoritmoEnvelopado + `"></Transform>`)
	b.WriteString(`<Transform Algorithm="` + AlgoritmoC14N + `"></Transform>`)
	b.WriteString(`</Transforms>`)
	b.WriteString(`<DigestMethod Algorithm="` + AlgoritmoResumo + `"></DigestMethod>`)
	b.WriteString(`<DigestValue>` + resumoB64 + `</DigestValue>`)
	b.WriteString(`</Reference>`)
	b.WriteString(`</SignedInfo>`)
	return b.String()
}

var escapadorAtributo = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	`"`, "&quot;",
	"\t", "&#x9;",
	"\n", "&#xA;",
	"\r", "&#xD;",
)

func escaparAtributo(s string) string { return escapadorAtributo.Replace(s) }

// Verificar confere a assinatura de um documento: recalcula o resumo do
// elemento referenciado, refaz o cálculo sobre o SignedInfo e valida a
// assinatura com a chave pública do certificado embutido.
//
// A verificação é criptográfica e estrutural. Ela não estabelece confiança na
// cadeia de certificação nem consulta listas de revogação — quem precisa disso
// deve validar [Certificado] contra as raízes da ICP-Brasil separadamente.
func Verificar(documento []byte) error {
	doc, err := xmldom.Ler(documento)
	if err != nil {
		return err
	}

	assinatura := doc.Raiz.Buscar("Signature", EspacoAssinatura)
	if assinatura == nil {
		return ErrSemAssinatura
	}
	signedInfo := assinatura.Filho("SignedInfo")
	if signedInfo == nil {
		return fmt.Errorf("%w: Signature sem SignedInfo", ErrAssinaturaInvalida)
	}

	if err := conferirAlgoritmos(signedInfo); err != nil {
		return err
	}

	referencia := signedInfo.Filho("Reference")
	if referencia == nil {
		return fmt.Errorf("%w: SignedInfo sem Reference", ErrAssinaturaInvalida)
	}
	uri, _ := referencia.Atributo("URI")
	id := strings.TrimPrefix(uri, "#")
	if id == "" {
		return fmt.Errorf("%w: Reference sem URI", ErrAssinaturaInvalida)
	}

	alvo := elementoPorId(doc.Raiz, id)
	if alvo == nil {
		return fmt.Errorf("%w: nenhum elemento com Id=%q", ErrAssinaturaInvalida, id)
	}

	noDigest := referencia.Filho("DigestValue")
	if noDigest == nil {
		return fmt.Errorf("%w: Reference sem DigestValue", ErrAssinaturaInvalida)
	}
	esperado := strings.TrimSpace(noDigest.Texto())
	canonico := xmldom.Canonicalizar(alvo, xmldom.OpcoesC14N{RemoverAssinaturas: true})
	calculado := sha1.Sum(canonico)
	if base64.StdEncoding.EncodeToString(calculado[:]) != esperado {
		return fmt.Errorf("%w: o resumo de <%s> não confere com o DigestValue", ErrAssinaturaInvalida, alvo.NomeQualificado())
	}

	cert, err := certificadoDe(assinatura)
	if err != nil {
		return err
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: chave pública %T não é RSA", ErrAlgoritmoNaoSuportado, cert.PublicKey)
	}

	noValor := assinatura.Filho("SignatureValue")
	if noValor == nil {
		return fmt.Errorf("%w: Signature sem SignatureValue", ErrAssinaturaInvalida)
	}
	valor, err := base64.StdEncoding.DecodeString(semEspacos(noValor.Texto()))
	if err != nil {
		return fmt.Errorf("%w: SignatureValue não é base64 válido: %w", ErrAssinaturaInvalida, err)
	}

	canonicoSignedInfo := xmldom.Canonicalizar(signedInfo, xmldom.OpcoesC14N{})
	resumoSignedInfo := sha1.Sum(canonicoSignedInfo)
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA1, resumoSignedInfo[:], valor); err != nil {
		return fmt.Errorf("%w: %w", ErrAssinaturaInvalida, err)
	}
	return nil
}

func conferirAlgoritmos(signedInfo *xmldom.No) error {
	esperados := map[string]string{
		"CanonicalizationMethod": AlgoritmoC14N,
		"SignatureMethod":        AlgoritmoAssinatura,
	}
	for tag, esperado := range esperados {
		no := signedInfo.Filho(tag)
		if no == nil {
			return fmt.Errorf("%w: SignedInfo sem %s", ErrAssinaturaInvalida, tag)
		}
		if alg, _ := no.Atributo("Algorithm"); alg != esperado {
			return fmt.Errorf("%w: %s usa %q, esperado %q", ErrAlgoritmoNaoSuportado, tag, alg, esperado)
		}
	}
	referencia := signedInfo.Filho("Reference")
	if referencia == nil {
		return fmt.Errorf("%w: SignedInfo sem Reference", ErrAssinaturaInvalida)
	}
	metodo := referencia.Filho("DigestMethod")
	if metodo == nil {
		return fmt.Errorf("%w: Reference sem DigestMethod", ErrAssinaturaInvalida)
	}
	if alg, _ := metodo.Atributo("Algorithm"); alg != AlgoritmoResumo {
		return fmt.Errorf("%w: DigestMethod usa %q, esperado %q", ErrAlgoritmoNaoSuportado, alg, AlgoritmoResumo)
	}
	return nil
}

// Certificado extrai o certificado X.509 embutido na assinatura do documento.
func Certificado(documento []byte) (*x509.Certificate, error) {
	doc, err := xmldom.Ler(documento)
	if err != nil {
		return nil, err
	}
	assinatura := doc.Raiz.Buscar("Signature", EspacoAssinatura)
	if assinatura == nil {
		return nil, ErrSemAssinatura
	}
	return certificadoDe(assinatura)
}

func certificadoDe(assinatura *xmldom.No) (*x509.Certificate, error) {
	no := assinatura.Buscar("X509Certificate", EspacoAssinatura)
	if no == nil {
		return nil, fmt.Errorf("%w: assinatura sem X509Certificate", ErrAssinaturaInvalida)
	}
	der, err := base64.StdEncoding.DecodeString(semEspacos(no.Texto()))
	if err != nil {
		return nil, fmt.Errorf("%w: X509Certificate não é base64 válido: %w", ErrAssinaturaInvalida, err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("%w: X509Certificate não é um certificado válido: %w", ErrAssinaturaInvalida, err)
	}
	return cert, nil
}

// EstaAssinado informa se o documento contém um elemento Signature.
func EstaAssinado(documento []byte) bool {
	doc, err := xmldom.Ler(documento)
	if err != nil {
		return false
	}
	return doc.Raiz.Buscar("Signature", EspacoAssinatura) != nil
}

func elementoPorId(raiz *xmldom.No, id string) *xmldom.No {
	if raiz.Tipo == xmldom.TipoElemento {
		if v, ok := raiz.Atributo("Id"); ok && v == id {
			return raiz
		}
	}
	for _, f := range raiz.Filhos {
		if achado := elementoPorId(f, id); achado != nil {
			return achado
		}
	}
	return nil
}

func semEspacos(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, s)
}
