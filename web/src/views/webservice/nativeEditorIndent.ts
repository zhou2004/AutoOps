export interface TextEdit {
  from: number;
  to: number;
  text: string;
  selectionStart: number;
  selectionEnd: number;
}

/** Indent the selected lines without normalizing the rest of a native file. */
export function indentConfiguration(
  value: string,
  start: number,
  end: number,
  outdent = false,
  width = 4
): TextEdit | undefined {
  const lineStart = start > 0 ? value.lastIndexOf("\n", start - 1) + 1 : 0;
  if (start === end && !outdent) {
    let column = 0;
    for (const character of value.slice(lineStart, start))
      column += character === "\t" ? width - (column % width) : 1;
    const text = " ".repeat(width - (column % width));
    return {
      from: start,
      to: end,
      text,
      selectionStart: start + text.length,
      selectionEnd: start + text.length
    };
  }

  // A selection ending at the beginning of a line must not edit that next line.
  const lastCharacter = end > start ? end - 1 : end;
  const newline = value.indexOf("\n", lastCharacter);
  const lineEnd = newline < 0 ? value.length : newline;
  const lines = value.slice(lineStart, lineEnd).split("\n");
  let offset = lineStart;
  let selectionStart = start;
  let selectionEnd = end;
  let changed = false;
  const text = lines
    .map(line => {
      const remove = outdent
        ? line.startsWith("\t")
          ? 1
          : Math.min(width, line.match(/^ */)![0].length)
        : 0;
      const prefix = outdent ? "" : " ".repeat(width);
      if (remove || prefix) changed = true;
      if (start >= offset)
        selectionStart += prefix.length - Math.min(remove, start - offset);
      if (end >= offset)
        selectionEnd += prefix.length - Math.min(remove, end - offset);
      offset += line.length + 1;
      return prefix + line.slice(remove);
    })
    .join("\n");
  return changed
    ? { from: lineStart, to: lineEnd, text, selectionStart, selectionEnd }
    : undefined;
}
