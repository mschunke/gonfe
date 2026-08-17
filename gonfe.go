// Package gonfe é a raiz de uma biblioteca para emissão de documentos fiscais
// eletrônicos brasileiros, seguindo os padrões da Receita Federal e das
// Secretarias de Fazenda estaduais.
//
// Este pacote não contém funcionalidade: serve de porta de entrada para a
// documentação. O código está nos subpacotes, cada um com uma
// responsabilidade:
//
//   - [github.com/mschunke/gonfe/nfe] — modelo de dados da NF-e e da NFC-e no
//     leiaute 4.00, cálculo de totais, validação e montagem de lote.
//   - [github.com/mschunke/gonfe/nfce] — QR Code e URLs de consulta da NFC-e.
//   - [github.com/mschunke/gonfe/sefaz] — endereços dos serviços por unidade da
//     federação e cliente SOAP com autenticação mútua TLS.
//   - [github.com/mschunke/gonfe/xmldsig] — assinatura e verificação no perfil
//     de XML Signature da SEFAZ.
//   - [github.com/mschunke/gonfe/certificado] — certificados A1 em PKCS#12.
//   - [github.com/mschunke/gonfe/chave] — chave de acesso de 44 dígitos.
//   - [github.com/mschunke/gonfe/tipos] — decimal de precisão fixa, data e
//     data/hora nos formatos do leiaute.
//   - [github.com/mschunke/gonfe/validacao] — CPF, CNPJ e inscrição estadual.
//   - [github.com/mschunke/gonfe/uf] — códigos do IBGE, nomes e fusos horários.
//
// # Emissão em cinco passos
//
//	cert, _ := certificado.CarregarArquivo("certificado.pfx", senha)
//
//	n := nfe.Nova(nfe.ModeloNFe)
//	// ... preencher identificação, emitente, destinatário, itens e pagamento
//
//	n.Preparar()            // normaliza, calcula totais e monta a chave
//	n.Validar()             // confere antes de gastar uma ida à SEFAZ
//	assinada, _ := n.AssinarCom(cert)
//
//	cliente, _ := sefaz.NovoCliente(sefaz.Config{
//		UF: uf.RS, Ambiente: nfe.Homologacao,
//		Modelo: nfe.ModeloNFe, Certificado: cert,
//	})
//	lote, _ := nfe.MontarLote("1", false, assinada)
//	envio, _ := cliente.Autorizar(ctx, lote)
//	resultado, _ := cliente.EsperarProcessamento(ctx, envio.Recibo(), 3*time.Second, 20)
//	proc, _ := nfe.MontarNFeProc(assinada, resultado.ProtocoloDa(n.Chave()))
//
// O guia completo está em https://mschunke.github.io/gonfe/ e os exemplos
// executáveis, no diretório exemplos do repositório.
//
// # Aviso legal
//
// Este projeto não tem vínculo com a Receita Federal do Brasil nem com nenhuma
// Secretaria de Fazenda estadual. A responsabilidade pelos documentos emitidos
// é de quem os emite.
package gonfe

// Versao é a versão desta biblioteca.
const Versao = "0.1.0"

// VersaoLeiaute é a versão do leiaute dos documentos fiscais implementada.
const VersaoLeiaute = "4.00"
