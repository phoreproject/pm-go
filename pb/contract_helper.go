package pb

import (
	"encoding/hex"
	"fmt"

<<<<<<< HEAD
	"github.com/phoreproject/btcutil"
	"github.com/phoreproject/wallet-interface"
=======
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcutil"
>>>>>>> 1eba569e5bc08b0e8756887aa5838fee26022b3c
)

func DisputeResolutionPayoutOutputToAddress(w wallet.Wallet, output *DisputeResolution_Payout_Output) (btcutil.Address, error) {
	var (
		addr btcutil.Address
		err  error
	)
	switch output.ScriptOrAddress.(type) {
	case *DisputeResolution_Payout_Output_Script:
		scriptBytes, err := hex.DecodeString(output.GetScript())
		if err != nil {
			return nil, err
		}
		addr, err = w.ScriptToAddress(scriptBytes)
		if err != nil {
			return nil, err
		}
	case *DisputeResolution_Payout_Output_Address:
		addr, err = w.DecodeAddress(output.GetAddress())
		if err != nil {
			return nil, fmt.Errorf("decoding dispute payout address: %s", err.Error())
		}
	case nil:
		return nil, fmt.Errorf("dispute resolution payout output has no destination set")
	default:
		return nil, fmt.Errorf("dispute resolution payout output contains an unexpected type %T", output.ScriptOrAddress)
	}
	return addr, nil
}
