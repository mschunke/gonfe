package nfce

import (
	"fmt"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/uf"
)

// Os endereços de consulta e de QR Code da NFC-e são definidos por cada estado
// e mudam com mais frequência do que os serviços web. A tabela abaixo reúne os
// endereços publicados pelas secretarias e serve como padrão de conveniência.
//
// Antes de emitir em produção, confira os endereços da sua UF no Portal da NF-e
// e, se divergirem, informe-os em [Opcoes.URLQRCode] e [Opcoes.URLConsulta].
// Um endereço errado não impede a autorização da nota, mas gera um cupom cujo
// QR Code o consumidor não consegue consultar.

type parURL struct {
	producao    string
	homologacao string
}

var urlsQRCode = map[uf.UF]parURL{
	uf.AC: {"http://www.sefaznet.ac.gov.br/nfce/qrcode", "http://hml.sefaznet.ac.gov.br/nfce/qrcode"},
	uf.AL: {"http://nfce.sefaz.al.gov.br/QRCode/consultarNFCe.jsp", "http://nfce.sefaz.al.gov.br/nfce-web/consultarNFCe"},
	uf.AM: {"http://sistemas.sefaz.am.gov.br/nfceweb/consultarNFCe.jsp", "http://homnfce.sefaz.am.gov.br/nfce-web/consultarNFCe"},
	uf.AP: {"https://www.sefaz.ap.gov.br/nfce/nfcep.php", "https://www.sefaz.ap.gov.br/nfce/nfcehml.php"},
	uf.BA: {"http://nfe.sefaz.ba.gov.br/servicos/nfce/qrcode.aspx", "http://hnfe.sefaz.ba.gov.br/servicos/nfce/qrcode.aspx"},
	uf.CE: {"http://nfce.sefaz.ce.gov.br/pages/ShowNFCe.html", "http://nfceh.sefaz.ce.gov.br/pages/ShowNFCe.html"},
	uf.DF: {"http://www.fazenda.df.gov.br/nfce/qrcode", "http://www.fazenda.df.gov.br/nfce/qrcode"},
	uf.ES: {"http://app.sefaz.es.gov.br/ConsultaNFCe/qrcode.aspx", "http://homologacao.sefaz.es.gov.br/ConsultaNFCe/qrcode.aspx"},
	uf.GO: {"http://nfe.sefaz.go.gov.br/nfeweb/sites/nfce/danfeNFCe", "http://homolog.sefaz.go.gov.br/nfeweb/sites/nfce/danfeNFCe"},
	uf.MA: {"http://www.nfce.sefaz.ma.gov.br/portal/consultarNFCe.jsp", "http://homologacao.sefaz.ma.gov.br/portal/consultarNFCe.jsp"},
	uf.MG: {"https://portalsped.fazenda.mg.gov.br/portalnfce/sistema/qrcode.xhtml", "https://hportalsped.fazenda.mg.gov.br/portalnfce/sistema/qrcode.xhtml"},
	uf.MS: {"http://www.dfe.ms.gov.br/nfce/qrcode", "http://www.dfe.ms.gov.br/nfce/qrcode"},
	uf.MT: {"http://www.sefaz.mt.gov.br/nfce/consultanfce", "http://homologacao.sefaz.mt.gov.br/nfce/consultanfce"},
	uf.PA: {"https://appnfc.sefa.pa.gov.br/portal/view/consultas/nfce/nfceForm.seam", "https://appnfc.sefa.pa.gov.br/portal-homologacao/view/consultas/nfce/nfceForm.seam"},
	uf.PB: {"http://www.receita.pb.gov.br/nfce", "http://www.receita.pb.gov.br/nfcehom"},
	uf.PE: {"http://nfce.sefaz.pe.gov.br/nfce/consulta", "http://nfcehomolog.sefaz.pe.gov.br/nfce/consulta"},
	uf.PI: {"http://www.sefaz.pi.gov.br/nfce/qrcode", "http://www.sefaz.pi.gov.br/nfce/qrcode"},
	uf.PR: {"http://www.fazenda.pr.gov.br/nfce/qrcode", "http://www.fazenda.pr.gov.br/nfce/qrcode"},
	uf.RJ: {"http://www4.fazenda.rj.gov.br/consultaNFCe/QRCode", "http://www4.fazenda.rj.gov.br/consultaNFCe/QRCode"},
	uf.RN: {"http://nfce.set.rn.gov.br/consultarNFCe.aspx", "http://hom.nfce.set.rn.gov.br/consultarNFCe.aspx"},
	uf.RO: {"http://www.nfce.sefin.ro.gov.br/consultanfce/consulta.jsp", "http://www.nfce.sefin.ro.gov.br/consultanfce/consulta.jsp"},
	uf.RR: {"https://www.sefaz.rr.gov.br/nfce/servlet/qrcode", "https://homolog.sefaz.rr.gov.br/nfce/servlet/qrcode"},
	uf.RS: {"https://www.sefaz.rs.gov.br/NFCE/NFCE-COM.aspx", "https://www.sefaz.rs.gov.br/NFCE/NFCE-COM.aspx"},
	uf.SC: {"https://sat.sef.sc.gov.br/nfce/consulta", "https://hom.sat.sef.sc.gov.br/nfce/consulta"},
	uf.SE: {"http://www.nfce.se.gov.br/portal/portalNoticias.jsp", "http://www.hom.nfe.se.gov.br/portal/portalNoticias.jsp"},
	uf.SP: {"https://www.nfce.fazenda.sp.gov.br/qrcode", "https://www.homologacao.nfce.fazenda.sp.gov.br/qrcode"},
	uf.TO: {"http://www.sefaz.to.gov.br/nfce/qrcode", "http://homologacao.sefaz.to.gov.br/nfce/qrcode"},
}

var urlsConsulta = map[uf.UF]parURL{
	uf.AC: {"http://www.sefaznet.ac.gov.br/nfce/consulta", "http://hml.sefaznet.ac.gov.br/nfce/consulta"},
	uf.AL: {"http://nfce.sefaz.al.gov.br/consultaNFCe.htm", "http://nfce.sefaz.al.gov.br/consultaNFCe.htm"},
	uf.AM: {"http://sistemas.sefaz.am.gov.br/nfceweb/formConsulta.do", "http://homnfce.sefaz.am.gov.br/nfce-web/formConsulta"},
	uf.AP: {"https://www.sefaz.ap.gov.br/nfce/nfcep.php", "https://www.sefaz.ap.gov.br/nfce/nfcehml.php"},
	uf.BA: {"http://nfe.sefaz.ba.gov.br/servicos/nfce/default.aspx", "http://hnfe.sefaz.ba.gov.br/servicos/nfce/default.aspx"},
	uf.CE: {"http://nfce.sefaz.ce.gov.br/pages/consultaNFCe.jsf", "http://nfceh.sefaz.ce.gov.br/pages/consultaNFCe.jsf"},
	uf.DF: {"http://www.fazenda.df.gov.br/nfce/consulta", "http://www.fazenda.df.gov.br/nfce/consulta"},
	uf.ES: {"http://app.sefaz.es.gov.br/ConsultaNFCe", "http://homologacao.sefaz.es.gov.br/ConsultaNFCe"},
	uf.GO: {"http://nfe.sefaz.go.gov.br/nfeweb/sites/nfce/consultanfce", "http://homolog.sefaz.go.gov.br/nfeweb/sites/nfce/consultanfce"},
	uf.MA: {"http://www.nfce.sefaz.ma.gov.br/portal/consultaNFCe.jsp", "http://homologacao.sefaz.ma.gov.br/portal/consultaNFCe.jsp"},
	uf.MG: {"https://portalsped.fazenda.mg.gov.br/portalnfce/sistema/consultaarg.xhtml", "https://hportalsped.fazenda.mg.gov.br/portalnfce/sistema/consultaarg.xhtml"},
	uf.MS: {"http://www.dfe.ms.gov.br/nfce/consulta", "http://www.dfe.ms.gov.br/nfce/consulta"},
	uf.MT: {"http://www.sefaz.mt.gov.br/nfce/consultanfce", "http://homologacao.sefaz.mt.gov.br/nfce/consultanfce"},
	uf.PA: {"https://appnfc.sefa.pa.gov.br/portal/view/consultas/nfce/consultanfce.seam", "https://appnfc.sefa.pa.gov.br/portal-homologacao/view/consultas/nfce/consultanfce.seam"},
	uf.PB: {"http://www.receita.pb.gov.br/nfce", "http://www.receita.pb.gov.br/nfcehom"},
	uf.PE: {"http://nfce.sefaz.pe.gov.br/nfce-web/consultarNFCe", "http://nfcehomolog.sefaz.pe.gov.br/nfce-web/consultarNFCe"},
	uf.PI: {"http://www.sefaz.pi.gov.br/nfce/consulta", "http://www.sefaz.pi.gov.br/nfce/consulta"},
	uf.PR: {"http://www.fazenda.pr.gov.br/nfce/consulta", "http://www.fazenda.pr.gov.br/nfce/consulta"},
	uf.RJ: {"http://www4.fazenda.rj.gov.br/consultaNFCe/QRCode", "http://www4.fazenda.rj.gov.br/consultaNFCe/QRCode"},
	uf.RN: {"http://nfce.set.rn.gov.br/consultarNFCe.aspx", "http://hom.nfce.set.rn.gov.br/consultarNFCe.aspx"},
	uf.RO: {"http://www.nfce.sefin.ro.gov.br/consultanfce/consulta.jsp", "http://www.nfce.sefin.ro.gov.br/consultanfce/consulta.jsp"},
	uf.RR: {"https://www.sefaz.rr.gov.br/nfce/servlet/wp_consulta_nfce", "https://homolog.sefaz.rr.gov.br/nfce/servlet/wp_consulta_nfce"},
	uf.RS: {"https://www.sefaz.rs.gov.br/NFCE/NFCE-COM.aspx", "https://www.sefaz.rs.gov.br/NFCE/NFCE-COM.aspx"},
	uf.SC: {"https://sat.sef.sc.gov.br/nfce/consulta", "https://hom.sat.sef.sc.gov.br/nfce/consulta"},
	uf.SE: {"http://www.nfce.se.gov.br/portal/consultarNFCe.jsp", "http://www.hom.nfe.se.gov.br/portal/consultarNFCe.jsp"},
	uf.SP: {"https://www.nfce.fazenda.sp.gov.br/consulta", "https://www.homologacao.nfce.fazenda.sp.gov.br/consulta"},
	uf.TO: {"http://www.sefaz.to.gov.br/nfce/consulta", "http://homologacao.sefaz.to.gov.br/nfce/consulta"},
}

// URLQRCode devolve o endereço base do QR Code da unidade da federação no
// ambiente informado.
func URLQRCode(unidade uf.UF, ambiente nfe.Ambiente) (string, error) {
	return buscarURL(urlsQRCode, unidade, ambiente, "QR Code")
}

// URLConsulta devolve o endereço de consulta da NFC-e pela chave de acesso.
func URLConsulta(unidade uf.UF, ambiente nfe.Ambiente) (string, error) {
	return buscarURL(urlsConsulta, unidade, ambiente, "consulta")
}

func buscarURL(tabela map[uf.UF]parURL, unidade uf.UF, ambiente nfe.Ambiente, servico string) (string, error) {
	par, ok := tabela[unidade]
	if !ok {
		return "", fmt.Errorf("nfce: não há endereço de %s cadastrado para %s; informe-o nas opções", servico, unidade)
	}
	switch ambiente {
	case nfe.Producao:
		return par.producao, nil
	case nfe.Homologacao:
		return par.homologacao, nil
	default:
		return "", fmt.Errorf("nfce: ambiente %q desconhecido", ambiente)
	}
}

// UFsComEndereco lista as unidades da federação para as quais a biblioteca
// conhece os endereços de QR Code e de consulta.
func UFsComEndereco() []uf.UF {
	lista := make([]uf.UF, 0, len(urlsQRCode))
	for _, u := range uf.Todas() {
		if _, ok := urlsQRCode[u]; ok {
			lista = append(lista, u)
		}
	}
	return lista
}
