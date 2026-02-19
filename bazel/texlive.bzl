"""Module extension that exposes the host pdflatex as a Bazel-managed tool.

A fully hermetic TeX Live download (~4GB) is impractical for this project.
Instead, this wraps the host installation so that:
  - `bazel build` fails fast with clear instructions if pdflatex is missing
  - The binary is available as @texlive//:pdflatex for use in `data` deps
  - The path is deterministic within the Bazel sandbox
"""

def _texlive_repo_impl(ctx):
    result = ctx.execute(["which", "pdflatex"], timeout = 5)
    if result.return_code != 0:
        ctx.file("BUILD.bazel", content = """\
# pdflatex not found — provide a stub that prints install instructions.
package(default_visibility = ["//visibility:public"])

exports_files(["pdflatex_missing.sh"])

alias(
    name = "pdflatex",
    actual = ":pdflatex_missing.sh",
)
""")
        ctx.file("pdflatex_missing.sh", content = """\
#!/bin/bash
echo "ERROR: pdflatex is not installed." >&2
echo "Install with:" >&2
echo "  macOS:  brew install --cask mactex-no-gui" >&2
echo "  Ubuntu: sudo apt-get install texlive-latex-base texlive-latex-extra" >&2
echo "  RHEL:   sudo yum install texlive texlive-latex" >&2
exit 1
""", executable = True)
        return

    host_pdflatex = result.stdout.strip()
    ctx.file("BUILD.bazel", content = """\
package(default_visibility = ["//visibility:public"])

exports_files(["pdflatex.sh"])

alias(
    name = "pdflatex",
    actual = ":pdflatex.sh",
)
""")
    ctx.file("pdflatex.sh", content = """\
#!/bin/bash
exec {pdflatex} "$@"
""".format(pdflatex = host_pdflatex), executable = True)

texlive_repo = repository_rule(
    implementation = _texlive_repo_impl,
    local = True,
    environ = ["PATH"],
    doc = "Wraps the host pdflatex as a Bazel target.",
)

def _texlive_ext_impl(ctx):
    texlive_repo(name = "texlive")

texlive = module_extension(
    implementation = _texlive_ext_impl,
)
