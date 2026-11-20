import { useLanguage } from '../../i18n/LanguageProvider'
import type { ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'

function extractText(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(extractText).join('')

  if (node && typeof node === 'object' && 'props' in node) {
    return extractText((node.props as { children?: ReactNode }).children)
  }

  return ''
}

async function copyText(value: string) {
  try {
    await navigator.clipboard.writeText(value)
  } catch {
    return
  }
}

export default function MarkdownContent({ content }: { content: string }) {
  const { t } = useLanguage()

  return (
    <div className="prose smoi-markdown">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        components={{
          a: ({ href, children }) => (
            <a href={href} target="_blank" rel="noopener noreferrer">
              {children}
              <span className="visually-hidden"> ({t('opensNewTab')})</span>
            </a>
          ),
          pre: ({ children }) => (
            <div className="smoi-code-container">
              <div className="smoi-code-toolbar">
                <span>{t('code')}</span>
                <button
                  className="btn btn-sm btn-outline-secondary"
                  type="button"
                  aria-label={t('copyCode')}
                  onClick={() => void copyText(extractText(children))}
                >
                  <i className="bi bi-clipboard me-1" aria-hidden="true" />{t('copy')}
                </button>
              </div>
              <pre>{children}</pre>
            </div>
          )
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
