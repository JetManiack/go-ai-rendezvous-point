import { renderMarkdown } from "./markdown.js";

export default function Markdown({ text }) {
  return <div className="markdown-body" dangerouslySetInnerHTML={renderMarkdown(text)} />;
}
