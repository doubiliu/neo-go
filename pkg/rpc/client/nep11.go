package client

import (
	"fmt"

	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/encoding/address"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
)

// NEP11Decimals invokes `decimals` NEP11 method on a specified contract.
func (c *Client) NEP11Decimals(tokenHash util.Uint160) (int64, error) {
	return c.nepDecimals(tokenHash)
}

// NEP11Symbol invokes `symbol` NEP11 method on a specified contract.
func (c *Client) NEP11Symbol(tokenHash util.Uint160) (string, error) {
	return c.nepSymbol(tokenHash)
}

// NEP11TotalSupply invokes `totalSupply` NEP11 method on a specified contract.
func (c *Client) NEP11TotalSupply(tokenHash util.Uint160) (int64, error) {
	return c.nepTotalSupply(tokenHash)
}

// NEP11BalanceOf invokes `balanceOf` NEP11 method on a specified contract.
func (c *Client) NEP11BalanceOf(tokenHash, owner util.Uint160) (int64, error) {
	return c.nepBalanceOf(tokenHash, owner, nil)
}

// TODO: support NEP11TokensOf? (it returns iterator)

// TransferNEP11 creates an invocation transaction that invokes 'transfer' method
// on a given token to move the whole NEP11 token with the specified token ID to
// given account and sends it to the network returning just a hash of it.
func (c *Client) TransferNEP11(acc *wallet.Account, to util.Uint160,
	tokenHash util.Uint160, tokenID string, gas int64) (util.Uint256, error) {
	tx, err := c.createNEP11TransferTx(acc, tokenHash, gas, to, tokenID)
	if err != nil {
		return util.Uint256{}, err
	}

	if err := acc.SignTx(tx); err != nil {
		return util.Uint256{}, fmt.Errorf("can't sign NEP11 transfer tx: %w", err)
	}

	return c.SendRawTransaction(tx)
}

// createNEP11TransferTx is an internal helper for TransferNEP11 and
// TransferNEP11Divisible which creates an invocation transaction for the
// 'transfer' method of a given contract (token) to move the whole (or the
// specified amount of) NEP11 token with the specified token ID to given account
// and returns it. The returned transaction is not signed.
// `args` for TransferNEP11:  to util.Uint160, tokenID string;
// `args` for TransferNEP11Divisible: from, to util.Uint160, amount int64, tokenID string.
func (c *Client) createNEP11TransferTx(acc *wallet.Account, tokenHash util.Uint160,
	gas int64, args ...interface{}) (*transaction.Transaction, error) {
	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter, tokenHash, "transfer", callflag.All, args...)
	emit.Opcodes(w.BinWriter, opcode.ASSERT)
	if w.Err != nil {
		return nil, fmt.Errorf("failed to create NEP11 transfer script: %w", w.Err)
	}
	from, err := address.StringToUint160(acc.Address)
	if err != nil {
		return nil, fmt.Errorf("bad account address: %w", err)
	}
	return c.CreateTxFromScript(w.Bytes(), acc, -1, gas, []SignerAccount{{
		Signer: transaction.Signer{
			Account: from,
			Scopes:  transaction.CalledByEntry,
		},
		Account: acc,
	}})
}

// Non-divisible NFT methods section start.

// NEP11OwnerOfNonDivisible invokes `ownerOf` non-devisible NEP11 method with the
// specified token ID on a specified contract.
func (c *Client) NEP11OwnerOfNonDivisible(tokenHash util.Uint160, tokenID string) (util.Uint160, error) {
	result, err := c.InvokeFunction(tokenHash, "ownerOf", []smartcontract.Parameter{
		{
			Type:  smartcontract.StringType,
			Value: tokenID,
		},
	}, nil)
	if err != nil {
		return util.Uint160{}, err
	}
	err = getInvocationError(result)
	if err != nil {
		return util.Uint160{}, err
	}

	return topUint160FromStack(result.Stack)
}

// Non-divisible NFT methods section end.

// Divisible NFT methods section start.

// TransferNEP11Divisible creates an invocation transaction that invokes 'transfer'
// method on a given token to move specified amount of divisible NEP11 assets
// (in FixedN format using contract's number of decimals) to given account and
// sends it to the network returning just a hash of it.
func (c *Client) TransferNEP11Divisible(acc *wallet.Account, to util.Uint160,
	tokenHash util.Uint160, amount int64, tokenID string, gas int64) (util.Uint256, error) {
	from, err := address.StringToUint160(acc.Address)
	if err != nil {
		return util.Uint256{}, fmt.Errorf("bad account address: %w", err)
	}
	tx, err := c.createNEP11TransferTx(acc, tokenHash, gas, acc.Address, from, to, amount, tokenID)
	if err != nil {
		return util.Uint256{}, err
	}

	if err := acc.SignTx(tx); err != nil {
		return util.Uint256{}, fmt.Errorf("can't sign NEP11 divisible transfer tx: %w", err)
	}

	return c.SendRawTransaction(tx)
}

// TODO: support NEP11OwnerOf? (it returns iterator)

// NEP11BalanceOfDivisible invokes `balanceOf` divisible NEP11 method on a
// specified contract.
func (c *Client) NEP11BalanceOfDivisible(tokenHash, owner util.Uint160, tokenID string) (int64, error) {
	return c.nepBalanceOf(tokenHash, owner, &tokenID)
}

// Divisible NFT methods section end.

// Optional NFT methods section start.

// TODO: support Tokens? (it returns iterator)

// NEP11PropertiesOptional invokes `properties` optional NEP11 method on a
// specified contract.
func (c *Client) NEP11PropertiesOptional(tokenHash util.Uint160, tokenID string) (*stackitem.Map, error) {
	result, err := c.InvokeFunction(tokenHash, "properties", []smartcontract.Parameter{{
		Type:  smartcontract.StringType,
		Value: tokenID,
	}}, nil)
	if err != nil {
		return nil, err
	}
	err = getInvocationError(result)
	if err != nil {
		return nil, err
	}

	return topMapFromStack(result.Stack)
}

// Optional NFT methods section end.
