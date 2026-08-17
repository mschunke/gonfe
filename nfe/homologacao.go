package nfe

// TextoObrigatorioHomologacaoNFCe é a descrição que a SEFAZ espera no primeiro
// item de uma NFC-e emitida em ambiente de homologação.
//
// A exigência não é uniforme entre as unidades da federação — algumas a
// aplicam, outras não —, e por isso [NFe.Validar] não a impõe. Use
// [AjustarParaHomologacao] para deixar o campo preenchido de qualquer forma:
// nas UFs que não exigem, o texto é apenas uma descrição incomum de produto.
const TextoObrigatorioHomologacaoNFCe = "NOTA FISCAL EMITIDA EM AMBIENTE DE HOMOLOGACAO - SEM VALOR FISCAL"

// AjustarParaHomologacao aplica as exigências de conteúdo do ambiente de
// homologação, que variam conforme o modelo do documento:
//
//   - na NF-e modelo 55, a razão social do destinatário passa a ser
//     [TextoObrigatorioHomologacao];
//   - na NFC-e modelo 65, a descrição do primeiro item passa a ser
//     [TextoObrigatorioHomologacaoNFCe].
//
// A função não faz nada quando o ambiente da nota é produção, o que a torna
// segura de chamar sempre — inclusive em código que emite nos dois ambientes
// conforme a configuração.
//
//	n.InfNFe.Ide.TpAmb = ambienteConfigurado
//	nfe.AjustarParaHomologacao(n)
func AjustarParaHomologacao(n *NFe) {
	if n == nil || n.InfNFe.Ide.TpAmb != Homologacao {
		return
	}
	switch n.InfNFe.Ide.Mod {
	case ModeloNFe:
		if n.InfNFe.Dest != nil {
			n.InfNFe.Dest.XNome = TextoObrigatorioHomologacao
		}
	case ModeloNFCe:
		if len(n.InfNFe.Det) > 0 {
			n.InfNFe.Det[0].Prod.XProd = TextoObrigatorioHomologacaoNFCe
		}
	}
}
