package utest

import (
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
	"time"

	"github.com/tendermint/go-crypto"
)

const genesisStr = `{
  "chain_id": "test",
  "chain_version": "2",
  "genesis_time": "2019-04-25T21:29:56.2838591+08:00",
  "app_hash": "",
  "app_state": {
    "token": {
      "address": "testAiWusnsFsUQWkfHGSK5zh63FozCPV5PiB",
      "owner": "testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu",
      "version": "",
      "name": "TSC",
      "symbol": "TSC",
      "totalSupply": 2000000000000000000,
      "addSupplyEnabled": false,
      "burnEnabled": false,
      "gasPrice": 2500
    },
    "rewardStrategy": [
      {
        "name": "validators",
        "rewardPercent": "20.00",
        "address": ""
      },
      {
        "name": "r_d_team",
        "rewardPercent": "20.00",
        "address": "testYvBLo6Eh8vZKppJnqJGNFWRLzjmxsuWo"
      },
      {
        "name": "bonus",
        "rewardPercent": "30.00",
        "address": "testPdV2xqipzeKtx9fkRCkWaZ7rNB6sDLyoG"
      },
      {
        "name": "reserved",
        "rewardPercent": "30.00",
        "address": "testDfzucuVwnug5243aHb61vf2s9AqERtzBa"
      }
    ],
    "contracts": [
      {
        "name": "governance",
        "address": "test5ZXho3uL5ZSBWpH2ifnbsBaQcy553gX8F",
		"account": "test4cdqVp2d2ovcAdwGNknkJuLvV1pdt8RWr",
        "owner": "testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu",
        "version": "2.0",
        "codeHash": "e8258ca241bc1cdf58e48c8da0fe29d5ea0cee47b7f697cb660e3a408024370f",
        "effectHeight": 1,
        "loseHeight": 0,
        "token": "",
        "orgID": "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G",
        "methods": [
          {
            "methodId": "5dde7015",
            "prototype": "NewValidator(string,types.PubKey,types.Address,int64)",
            "gas": 50000
          },
          {
            "methodId": "ed21b83",
            "prototype": "SetPower(types.PubKey,int64)",
            "gas": 20000
          },
          {
            "methodId": "972d98bb",
            "prototype": "SetRewardAddr(types.PubKey,types.Address)",
            "gas": 20000
          },
          {
            "methodId": "6246ab67",
            "prototype": "SetRewardStrategy(string)",
            "gas": 50000
          }
        ]
      },
      {
        "name": "organization",
        "address": "testARRVfZyZijgkuxzA5VZFhho651bFFgmgs",
		"account": "testJ7V1532c9T9MLdDBLzXXgBqrMKYuWnwVV",
        "owner": "testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu",
        "version": "2.0",
        "codeHash": "EEF61882AC72817B5C0AE24FFED64DA9C7B4552D31099EF38135991A7D8E2FD5",
        "effectHeight": 1,
        "loseHeight": 0,
        "token": "",
        "orgID": "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G",
        "methods": [
          {
            "methodId": "9e922f48",
            "prototype": "RegisterOrganization(string)string",
            "gas": 500000
          },
          {
            "methodId": "62191292",
            "prototype": "SetSigners(string,[]types.PubKey)",
            "gas": 500000
          }
        ]
      },
      {
        "name": "smartcontract",
        "address": "test6ryVKSgdZBgcgR3aWgcmdZBBKXJKh1QJz",
		"account": "testGFMumw2caaLuSkFgTrYfdFM88DocKn13E",
        "owner": "testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu",
        "version": "2.0",
        "codeHash": "31C9E848545766E02CD715D32FADB50EB5A026C6221C9117DABA78D3ED86ADD3",
        "effectHeight": 1,
        "loseHeight": 0,
        "token": "",
        "orgID": "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G",
        "methods": [
          {
            "methodId": "d7596e75",
            "prototype": "Authorize(types.Address,string)",
            "gas": 50000
          },
          {
            "methodId": "e0da7827",
            "prototype": "DeployContract(string,string,string,types.Hash,[]byte,string,string,int64,types.Address)types.Address",
            "gas": 50000
          },
          {
            "methodId": "45385372",
            "prototype": "ForbidContract(types.Address,int64)",
            "gas": 50000
          }
        ]
      },
      {
        "name": "token-basic",
        "address": "testAiWusnsFsUQWkfHGSK5zh63FozCPV5PiB",
		"account": "testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu",
        "owner": "testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu",
        "version": "2.0",
        "codeHash": "563FAB3586B75D6831D313A14F45A1C23ABEB39B891D9FD726495EFF3A62E07A",
        "effectHeight": 1,
        "loseHeight": 0,
        "token": "testAiWusnsFsUQWkfHGSK5zh63FozCPV5PiB",
        "orgID": "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G",
        "methods": [
          {
            "methodId": "44d8ca60",
            "prototype": "Transfer(types.Address,bn.Number)",
            "gas": 500
          },
          {
            "methodId": "9024dc9b",
            "prototype": "SetGasPrice(int64)",
            "gas": 2000
          },
          {
            "methodId": "8856b980",
            "prototype": "SetBaseGasPrice(int64)",
            "gas": 2000
          }
        ],
		"interfaces": [
		  {
			"methodId": "44d8ca60",
            "prototype": "Transfer(types.Address,bn.Number)",
            "gas": 450
		  }
		]
      },
      {
        "name": "token-issue",
        "address": "testK6kjgCkABnSTS1MB86CixEQWM96ZbsP9",
		"account": "testBCbvLWS5fEkqoanfUDkgGw38mXQx5LXke",
        "owner": "testNKP3VFEniXL1kq36HuXuGaCGjJMazKhPu",
        "version": "2.0",
        "codeHash": "8917B2698F0ABD28F00D1881DE58E075BD0FADB7F332E1A36769C71143ED84B6",
        "effectHeight": 1,
        "loseHeight": 0,
        "token": "",
        "orgID": "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G",
        "methods": [
          {
            "methodId": "ed1d1d9a",
            "prototype": "NewToken(string,string,bn.Number,bool,bool,int64)types.Address",
            "gas": 20000
          },
          {
            "methodId": "44d8ca60",
            "prototype": "Transfer(types.Address,bn.Number)",
            "gas": 600
          },
          {
            "methodId": "1b0731b9",
            "prototype": "BatchTransfer([]types.Address,bn.Number)",
            "gas": 6000
          },
          {
            "methodId": "6b7e4ed5",
            "prototype": "AddSupply(bn.Number)",
            "gas": 2400
          },
          {
            "methodId": "fbbd9dd3",
            "prototype": "Burn(bn.Number)",
            "gas": 2400
          },
          {
            "methodId": "810b995f",
            "prototype": "SetOwner(types.Address)",
            "gas": 2400
          },
          {
            "methodId": "9024dc9b",
            "prototype": "SetGasPrice(int64)",
            "gas": 2400
          }
        ],
		"interfaces": [
		  {
			"methodId": "44d8ca60",
            "prototype": "Transfer(types.Address,bn.Number)",
            "gas": 540
		  }
		]
      },
      {
        "name": "myplayerbook",
        "address": "testWkNWzXyqMmumfxfXva2QV1qKa3aroVyu",
        "account": "test2uGLmMnsHauRUjyjQKGdXchUxpRMM8oeD",
        "owner": "testBsHvWxKkScTSpkF5gPFhrWegN2yosrZV9",
        "version": "2.0",
        "codeHash": "43A15EC506F3864126E78FD3E1A265D9EAF5D436E776C9B0200D77E57B76B7ED",
        "effectHeight": 1,
        "loseHeight": 0,
        "token": "",
        "orgID": "orgJgaGConUyK81zibntUBjQ33PKctpk1K1G",
        "methods": [
          {
            "methodId": "e463fdb2",
            "prototype": "RegisterName(string)(types.Error)",
            "gas": 500
          }
        ]
      }
    ]
  },
  "validators": [
    {
      "name": "node1",
      "reward_addr": "test8LtT8AonWgJ8nMCEdAR5UGrbRfUmuoeiz",
      "power": 10
    }
  ],
  "mode": 1
}
`

// GenesisValidator validator's info
type GenesisValidator struct {
	RewardAddr string        `json:"reward_addr"`
	PubKey     crypto.PubKey `json:"pub_key,omitempty"` // No Key In Genesis File,so omit empty
	Power      int64         `json:"power"`
	Name       string        `json:"name"`
}

// TokenContract token contract
type TokenContract struct {
	Address      types.Address `json:"address"`      // 合约地址
	EffectHeight int64         `json:"effectHeight"` // 合约生效的区块高度
}

// AppState app state
type AppState struct {
	GnsToken     std.Token      `json:"token"`
	GnsContracts []std.Contract `json:"contracts"`
}

type genesis struct {
	GenesisTime  time.Time          `json:"genesis_time"`
	ChainID      string             `json:"chain_id"`
	Validators   []GenesisValidator `json:"validators"`
	AppHash      types.HexBytes     `json:"app_hash"`
	AppStateJSON AppState           `json:"app_state"`
	Mode         int                `json:"mode,omitempty"`
}
