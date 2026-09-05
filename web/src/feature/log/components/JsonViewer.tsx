import React, { Suspense } from 'react'
import { Skeleton } from '@/components/ui/skeleton'

// Keep the JSON viewer out of the initial bundle; logs are opened on demand.
const LazyJsonView = React.lazy(() => import('@uiw/react-json-view'))

interface JsonViewerProps {
    src: unknown
    name?: string | false
    collapsed?: boolean | number
    enableClipboard?: boolean
    displayDataTypes?: boolean
    displayObjectSize?: boolean
    collapseStringsAfterLength?: number
    fallbackToRawText?: boolean
}

export function JsonViewer({
    src,
    name = false,
    collapsed = 2,
    enableClipboard = true,
    displayDataTypes = false,
    displayObjectSize = false,
    collapseStringsAfterLength = 100,
    fallbackToRawText = false,
}: JsonViewerProps) {
    let parsedSrc = src
    let shouldRenderRawText = false

    // 尝试解析字符串形式的JSON
    if (typeof src === 'string') {
        try {
            parsedSrc = JSON.parse(src)
        } catch {
            if (fallbackToRawText) {
                shouldRenderRawText = true
            } else {
                parsedSrc = src
            }
        }
    }

    if (shouldRenderRawText) {
        return (
            <pre
                className="overflow-x-auto whitespace-pre-wrap break-all rounded-md border bg-transparent p-2 text-[13px]"
                style={{
                    fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
                }}
            >
                {String(src)}
            </pre>
        )
    }

    return (
        <div className="json-viewer-container">
            <Suspense fallback={<Skeleton className="h-20 w-full" />}>
                <LazyJsonView
                    value={parsedSrc as object}
                    keyName={name || undefined}
                    collapsed={collapsed}
                    enableClipboard={enableClipboard}
                    displayDataTypes={displayDataTypes}
                    displayObjectSize={displayObjectSize}
                    shortenTextAfterLength={collapseStringsAfterLength}
                    style={{
                        backgroundColor: 'transparent', fontSize: '13px', fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace', padding: '8px',
                    }}
                />
            </Suspense>
        </div>
    )
} 
