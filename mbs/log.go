package mbs

func (b *Builder) logCommandOutput(d []byte) {
	if b.Options.LogOutput {
		b.Options.Stdout.Write(d)
	}
}

func (b *Builder) logVerbose(dag *target) {}
func (b *Builder) logError(dag *target)   {}
