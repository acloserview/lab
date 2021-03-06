## lab completion

Generates the shell autocompletion [bash, elvish, fish, powershell, xonsh, zsh]

### Synopsis

Generates the shell autocompletion [bash, elvish, fish, powershell, xonsh, zsh]

Scripts can be directly sourced (though using pre-generated versions is recommended to avoid shell startup delay):
  bash       : source <(lab completion)
  elvish     : eval(lab completion|slurp)
  fish       : lab completion fish | source
  powershell : lab completion | Out-String | Invoke-Expression
  xonsh      : exec($(lab completion))
  zsh        : source <(lab completion)

```
lab completion [shell] [flags]
```

### Options

```
  -h, --help   help for completion
```

### SEE ALSO

* [lab](index.md)	 - A Git Wrapper for GitLab

###### Auto generated by spf13/cobra on 24-Jan-2021
