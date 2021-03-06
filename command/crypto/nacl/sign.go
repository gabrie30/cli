package nacl

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/smallstep/cli/errs"
	"github.com/smallstep/cli/utils"
	"github.com/urfave/cli"
	"golang.org/x/crypto/nacl/sign"
)

func signCommand() cli.Command {
	return cli.Command{
		Name:      "sign",
		Usage:     "signs small messages using public-key cryptography",
		UsageText: "step crypto nacl sign <subcommand> [arguments] [global-flags] [subcommand-flags]",
		Description: `
**step crypto nacl sign** command group uses public-key cryptography to sign
and verify messages.

TODO

## EXAMPLES

Create a keypair for verifying amd signing messages:
'''
$ step crypto nacl sign keypair nacl.sign.pub nacl.sign.priv
'''

Sign a message using the private key:
'''
$ step crypto nacl sign sign nacl.sign.priv
Write text to sign: ********
rNrOfqsv4svlRnVPSVYe2REXodL78yEMHtNkzAGNp4MgHuVGoyayp0zx4D5rjTzYVVrD2HRP306ZILT62ohvCG1lc3NhZ2U

$ cat message.txt | step crypto nacl sign sign ~/step/keys/nacl.recipient.sign.priv
rNrOfqsv4svlRnVPSVYe2REXodL78yEMHtNkzAGNp4MgHuVGoyayp0zx4D5rjTzYVVrD2HRP306ZILT62ohvCG1lc3NhZ2U
'''

Verify the signed message using the public key:
'''
$ echo rNrOfqsv4svlRnVPSVYe2REXodL78yEMHtNkzAGNp4MgHuVGoyayp0zx4D5rjTzYVVrD2HRP306ZILT62ohvCG1lc3NhZ2U \
     | step crypto nacl sign open nacl.sign.pub
message
'''`,
		Subcommands: cli.Commands{
			signKeypairCommand(),
			signOpenCommand(),
			signSignCommand(),
		},
	}
}

func signKeypairCommand() cli.Command {
	return cli.Command{
		Name:      "keypair",
		Action:    cli.ActionFunc(signKeypairAction),
		Usage:     "generates a pair for use with sign and open",
		UsageText: "**step crypto nacl sign keypair** <pub-file> <priv-file>",
		Description: `**step crypto nacl sign keypair** generates a secret key and a corresponding
public key valid for verifying and signing messages.

For examples, see **step help crypto nacl sign**.`,
	}
}

func signOpenCommand() cli.Command {
	return cli.Command{
		Name:      "open",
		Action:    cli.ActionFunc(signOpenAction),
		Usage:     "verifies a signed message produced by sign",
		UsageText: "**step crypto nacl sign open** <pub-file>",
		Description: `**step crypto nacl sign open** verifies the signature of a message using the
signer's public key.

For examples, see **step help crypto nacl sign**.`,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "raw",
				Usage: "Indicates that input is not base64 encoded",
			},
		},
	}
}

func signSignCommand() cli.Command {
	return cli.Command{
		Name:      "sign",
		Action:    cli.ActionFunc(signSignAction),
		Usage:     "signs a message using Ed25519",
		UsageText: "**step crypto nacl sign sign** <priv-file>",
		Description: `**step crypto nacl sign keypair** signs a message m using the signer's private
key.

For examples, see **step help crypto nacl sign**.`,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "raw",
				Usage: "Do not base64 encode output",
			},
		},
	}
}

func signKeypairAction(ctx *cli.Context) error {
	if err := errs.NumberOfArguments(ctx, 2); err != nil {
		return err
	}

	args := ctx.Args()
	pubFile, privFile := args[0], args[1]
	if pubFile == privFile {
		return errs.EqualArguments(ctx, "<pub-file>", "<priv-file>")
	}

	pub, priv, err := sign.GenerateKey(rand.Reader)
	if err != nil {
		return errors.Wrap(err, "error generating key")
	}

	if err := utils.WriteFile(pubFile, pub[:], 0600); err != nil {
		return errs.FileError(err, pubFile)
	}

	if err := utils.WriteFile(privFile, priv[:], 0600); err != nil {
		return errs.FileError(err, privFile)
	}

	return nil
}

func signOpenAction(ctx *cli.Context) error {
	if err := errs.NumberOfArguments(ctx, 1); err != nil {
		return err
	}

	pubFile := ctx.Args().Get(0)
	pub, err := ioutil.ReadFile(pubFile)
	if err != nil {
		return errs.FileError(err, pubFile)
	} else if len(pub) != 32 {
		return errors.New("invalid public key: key size is not 32 bytes")
	}

	input, err := utils.ReadInput("Write signed message to open: ")
	if err != nil {
		return errors.Wrap(err, "error reading input")
	}

	var rawInput []byte
	if ctx.Bool("raw") {
		rawInput = input
	} else {
		// DecodeLen returns the maximum length,
		// Decode will return the actual length.
		rawInput = make([]byte, b64Encoder.DecodedLen(len(input)))
		n, err := b64Encoder.Decode(rawInput, input)
		if err != nil {
			return errors.Wrap(err, "error decoding base64 input")
		}
		rawInput = rawInput[:n]
	}

	var pb [32]byte
	copy(pb[:], pub)

	raw, ok := sign.Open(nil, rawInput, &pb)
	if !ok {
		return errors.New("error authenticating input")
	}

	os.Stdout.Write(raw)
	return nil
}

func signSignAction(ctx *cli.Context) error {
	if err := errs.NumberOfArguments(ctx, 1); err != nil {
		return err
	}

	privFile := ctx.Args().Get(0)
	priv, err := ioutil.ReadFile(privFile)
	if err != nil {
		return errs.FileError(err, privFile)
	} else if len(priv) != 64 {
		return errors.New("invalid private key: key size is not 64 bytes")
	}

	input, err := utils.ReadInput("Write text to sign: ")
	if err != nil {
		return errors.Wrap(err, "error reading input")
	}

	var pv [64]byte
	copy(pv[:], priv)

	raw := sign.Sign(nil, input, &pv)
	if ctx.Bool("raw") {
		os.Stdout.Write(raw)
	} else {
		fmt.Println(b64Encoder.EncodeToString(raw))
	}

	return nil
}
