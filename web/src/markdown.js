const MARKDOWN_ALLOWED_TAGS = [
  "p", "br", "strong", "em", "code", "pre",
  "h1", "h2", "h3",
  "ul", "ol", "li",
  "blockquote",
  "table", "thead", "tbody", "tr", "th", "td",
  "a",
];
const MARKDOWN_ALLOWED_ATTR = ["href"];

export function renderMarkdown(text) {
  const html = marked.parse(text);
  const clean = DOMPurify.sanitize(html, {
    ALLOWED_TAGS: MARKDOWN_ALLOWED_TAGS,
    ALLOWED_ATTR: MARKDOWN_ALLOWED_ATTR,
  });
  return { __html: clean };
}
