package config

import "testing"

func TestValidateKeep(t *testing.T) {
	tests := []struct {
		name    string
		keep    []string
		orders  []OrderList
		wantErr bool
	}{
		{
			name:    "no overlap",
			keep:    []string{"SOL", "BNB"},
			orders:  []OrderList{{Order: BuyCryptoMap{"BTCUSDT": 5, "ETHUSDT": 5}}},
			wantErr: false,
		},
		{
			name:    "kept asset in order list",
			keep:    []string{"SOL"},
			orders:  []OrderList{{Order: BuyCryptoMap{"SOLUSDT": 5}}},
			wantErr: true,
		},
		{
			name:    "kept asset in later order list",
			keep:    []string{"BNB"},
			orders: []OrderList{
				{Order: BuyCryptoMap{"BTCUSDT": 5}},
				{Order: BuyCryptoMap{"BNBUSDT": 5}},
			},
			wantErr: true,
		},
		{
			name:    "no keep configured",
			keep:    nil,
			orders:  []OrderList{{Order: BuyCryptoMap{"BTCUSDT": 5}}},
			wantErr: false,
		},
		{
			name:    "empty string in keep is ignored",
			keep:    []string{""},
			orders:  []OrderList{{Order: BuyCryptoMap{"BTCUSDT": 5}}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeep(tt.keep, tt.orders)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateKeep() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
