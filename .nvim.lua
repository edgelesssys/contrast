vim.lsp.config("gopls", {
	settings = {
		gopls = {
			gofumpt = true,
			buildFlags = { "-tags", "e2e,contrast_unstable_api" },
		},
	},
})
