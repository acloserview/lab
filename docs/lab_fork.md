## lab fork

Fork a remote repository on GitLab and add as remote

```
lab fork [remote|upstream-to-fork] [flags]
```

### Options

```
  -g, --group string   fork project in a different group (namespace)
  -h, --help           help for fork
      --http           fork using HTTP protocol instead of SSH
  -n, --name string    fork project with a different name
      --no-wait        don't wait for forking operation to finish
  -p, --path string    fork project with a different path
  -s, --skip-clone     skip clone after remote fork
```

### SEE ALSO

* [lab](index.md)	 - A Git Wrapper for GitLab

###### Auto generated by spf13/cobra on 24-Jan-2021
