package sefaz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mschunke/gonfe/dfe"
	"github.com/mschunke/gonfe/nfe"
)

// IntervaloEntreConsultasDFe é a espera recomendada entre duas chamadas
// consecutivas à distribuição de DF-e.
//
// A Receita bloqueia por uma hora quem consulta com frequência excessiva. Um
// minuto entre chamadas é o intervalo indicado quando ainda há fila a consumir;
// depois de esvaziar a fila, a orientação é consultar no máximo uma vez por
// hora.
const IntervaloEntreConsultasDFe = time.Minute

// DistribuicaoDFe consulta o serviço de distribuição do Ambiente Nacional uma
// única vez.
//
// A consulta vai sempre ao Ambiente Nacional, independentemente da UF do
// cliente; o código da UF entra na mensagem apenas para roteamento interno da
// Receita.
//
// Um bloqueio por consumo indevido devolve erro que casa com
// [dfe.ErrConsumoIndevido]. A fila vazia não é erro: a resposta volta com
// [dfe.Resposta.FilaVazia] verdadeiro.
func (c *Cliente) DistribuicaoDFe(ctx context.Context, consulta dfe.Consulta) (*dfe.Resposta, error) {
	documento, tipo := c.documentoDoConsulente()
	if documento == "" {
		return nil, errors.New("sefaz: a distribuição de DF-e exige o CNPJ ou o CPF do consulente; " +
			"informe um certificado ou preencha Config.CNPJConsulente")
	}

	var cnpj, cpf string
	if tipo == "CNPJ" {
		cnpj = documento
	} else {
		cpf = documento
	}

	mensagem, err := dfe.MontarConsulta(c.cfg.Ambiente, c.cfg.UF.Codigo(), cnpj, cpf, consulta)
	if err != nil {
		return nil, err
	}

	endereco, err := c.urlDeDistribuicao()
	if err != nil {
		return nil, err
	}

	var resposta dfe.Resposta
	if err := c.chamarEm(ctx, endereco, ServicoDistribuicaoDFe, mensagem, &resposta); err != nil {
		return nil, err
	}
	if resposta.CStat == dfe.StatusConsumoIndevido {
		return &resposta, fmt.Errorf("%w: %s", dfe.ErrConsumoIndevido, resposta.XMotivo)
	}
	if err := ErroDeStatus(ServicoDistribuicaoDFe, resposta.CStat, resposta.XMotivo,
		dfe.StatusComDocumentos, dfe.StatusSemDocumentos); err != nil {
		return &resposta, err
	}
	return &resposta, nil
}

// ConsumirDFe percorre a fila de distribuição a partir do NSU informado até
// esvaziá-la, entregando cada documento à função recebida.
//
// A função é chamada uma vez por documento, na ordem da fila, junto com o NSU
// já formatado. Devolver erro interrompe o consumo — e o NSU devolvido por
// ConsumirDFe passa a ser o do último documento processado com sucesso, para
// que a retomada não pule nada.
//
// Entre uma consulta e a seguinte é respeitado [IntervaloEntreConsultasDFe],
// porque consultar depressa demais faz a Receita bloquear o CNPJ por uma hora.
// Guarde o NSU devolvido: é dele que a próxima execução deve partir.
func (c *Cliente) ConsumirDFe(ctx context.Context, ultimoNSU string, aoReceber func(dfe.Documento) error) (string, error) {
	nsu := dfe.FormatarNSU(ultimoNSU)

	for volta := 0; ; volta++ {
		if volta > 0 {
			select {
			case <-ctx.Done():
				return nsu, ctx.Err()
			case <-time.After(IntervaloEntreConsultasDFe):
			}
		}

		resposta, err := c.DistribuicaoDFe(ctx, dfe.Consulta{UltimoNSU: nsu})
		if err != nil {
			return nsu, err
		}

		documentos, err := resposta.Documentos()
		if err != nil {
			return nsu, err
		}
		for _, d := range documentos {
			if err := aoReceber(d); err != nil {
				return nsu, fmt.Errorf("sefaz: processamento do NSU %s: %w", d.NSU, err)
			}
			nsu = d.NSU
		}

		// O ultNSU da resposta pode avançar além do último documento entregue,
		// quando a fila contém documentos que não interessam ao consulente.
		if avanco := dfe.FormatarNSU(resposta.UltNSU); avanco > nsu {
			nsu = avanco
		}
		if resposta.Fim() {
			return nsu, nil
		}
	}
}

// urlDeDistribuicao resolve o endereço do serviço de distribuição, que existe
// apenas no Ambiente Nacional.
func (c *Cliente) urlDeDistribuicao() (string, error) {
	if endereco, ok := c.cfg.Endpoints[ServicoDistribuicaoDFe]; ok && endereco != "" {
		return endereco, nil
	}
	return URL(AutorizadorAN, nfe.ModeloNFe, c.cfg.Ambiente, ServicoDistribuicaoDFe)
}

// documentoDoConsulente devolve o CNPJ ou o CPF de quem consulta, preferindo o
// que estiver na configuração e recorrendo ao certificado.
func (c *Cliente) documentoDoConsulente() (documento, tipo string) {
	if c.cfg.CNPJConsulente != "" {
		return c.cfg.CNPJConsulente, "CNPJ"
	}
	if c.cfg.CPFConsulente != "" {
		return c.cfg.CPFConsulente, "CPF"
	}
	if c.cfg.Certificado == nil {
		return "", ""
	}
	if cnpj := c.cfg.Certificado.CNPJ(); cnpj != "" {
		return cnpj, "CNPJ"
	}
	if cpf := c.cfg.Certificado.CPF(); cpf != "" {
		return cpf, "CPF"
	}
	return "", ""
}
