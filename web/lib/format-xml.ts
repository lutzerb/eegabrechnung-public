// Client-side pretty-printer for the EDA-Nachrichten XML preview. Outbound messages we build
// ourselves are already indented, but inbound XML is stored verbatim as received from the
// Netzbetreiber/edanet — often a single line with no whitespace at all — so the raw payload
// looked like an unreadable blob in the preview. Collapsing existing whitespace first makes
// this idempotent for already-formatted XML too, so both cases render identically.
export function formatXml(input: string): string {
  const trimmedInput = input.trim();
  if (!trimmedInput) return input;

  const withBreaks = trimmedInput.replace(/>\s*</g, ">\n<");
  const lines = withBreaks.split("\n");

  let indent = 0;
  const pad = "  ";
  const out: string[] = [];

  for (const raw of lines) {
    const line = raw.trim();
    if (!line) continue;

    const isClosingTag = line.startsWith("</");
    const isSelfClosing = /\/>$/.test(line);
    const isDeclOrComment = line.startsWith("<?") || line.startsWith("<!--");
    // An opening+content+matching-closing tag on one line (e.g. <ct:MessageAddress>RC999999</ct:MessageAddress>)
    // is net-zero for depth — don't indent its (non-existent) children.
    const isSelfContained = /^<([a-zA-Z_][\w:.-]*)(?:\s[^>]*)?>.*<\/\1>$/.test(line);

    if (isClosingTag) indent = Math.max(0, indent - 1);

    out.push(pad.repeat(indent) + line);

    if (!isClosingTag && !isSelfClosing && !isDeclOrComment && !isSelfContained) {
      indent++;
    }
  }

  return out.join("\n");
}
