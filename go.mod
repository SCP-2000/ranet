module github.com/SCP-2000/ranet

go 1.16

require (
	gitlab.com/NickCao/RAIT/v4 v4.2.0
	golang.zx2c4.com/wireguard v0.0.20201118
	gvisor.dev/gvisor v0.0.0-20210830223913-d93cb4e4ef6b
)

replace golang.zx2c4.com/wireguard/wgctrl => github.com/NickCao/wgctrl-go v0.0.0-20200721052646-81817b9b0823
