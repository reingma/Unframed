import type { Envelope } from "../../lib/documents";

// ArticleView renders the HTML body Go produced. Go owns Markdown processing
// and its HTML policy, so the body is inserted as is.
export function ArticleView({ envelope }: { envelope: Envelope }) {
  return (
    <>
      {envelope.metadata.draft && <p className="badge">Draft</p>}
      <article
        className="article"
        dangerouslySetInnerHTML={{ __html: envelope.body.content }}
      />
    </>
  );
}
