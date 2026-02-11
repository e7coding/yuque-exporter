package book

import "github.com/spf13/cobra"

func NewExporter() *cobra.Command {

	return &cobra.Command{
		Use:   "book",
		Short: "导出语雀知识库",
		Long:  "Create global unified access acceleration configuration item",
		Example: "ucloud pathx create --bandwidth 10 --area-code DXB" +
			"--charge-type Month --quantity 4 --accel Global --origin-ip 110.111.111.111" +
			"--protocol TCP --port 30654 --origin-port 30564",
	}
}
