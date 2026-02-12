'use client';

import { useEffect, useMemo } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { useOAuthTokenList } from '@/api/endpoints/oauth';
import { Card } from './Card';
import { usePaginationStore, useSearchStore } from '@/components/modules/toolbar';
import { EASING } from '@/lib/animations/fluid-transitions';
import { useGridPageSize } from '@/hooks/use-grid-page-size';

/** OAuth card height */
const OAUTH_CARD_HEIGHT = 180;

export function OAuth() {
    const { data: tokensData, isLoading } = useOAuthTokenList();
    const pageKey = 'oauth' as const;
    const pageSize = useGridPageSize({
        itemHeight: OAUTH_CARD_HEIGHT,
        gap: 16,
        columns: { default: 1, md: 2, lg: 3, xl: 4 },
    });
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const page = usePaginationStore((s) => s.getPage(pageKey));
    const setPage = usePaginationStore((s) => s.setPage);
    const setTotalItems = usePaginationStore((s) => s.setTotalItems);
    const setPageSize = usePaginationStore((s) => s.setPageSize);
    const direction = usePaginationStore((s) => s.getDirection(pageKey));

    const filteredTokens = useMemo(() => {
        if (!tokensData) return [];
        const sorted = [...tokensData].sort((a, b) => a.id - b.id);
        if (!searchTerm.trim()) return sorted;
        const term = searchTerm.toLowerCase();
        return sorted.filter((t) =>
            t.email?.toLowerCase().includes(term) ||
            t.type.toLowerCase().includes(term) ||
            t.remark?.toLowerCase().includes(term)
        );
    }, [tokensData, searchTerm]);

    useEffect(() => {
        setTotalItems(pageKey, filteredTokens.length);
        setPageSize(pageKey, pageSize);
    }, [filteredTokens.length, pageSize, pageKey, setTotalItems, setPageSize]);

    useEffect(() => {
        setPage(pageKey, 1);
    }, [searchTerm, pageKey, setPage]);

    const pagedTokens = useMemo(() => {
        const start = (page - 1) * pageSize;
        return filteredTokens.slice(start, start + pageSize);
    }, [filteredTokens, page, pageSize]);

    if (isLoading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
        );
    }

    return (
        <AnimatePresence mode="popLayout" initial={false} custom={direction}>
            <motion.div
                key={`oauth-page-${page}`}
                custom={direction}
                variants={{
                    enter: (d: number) => ({ x: d >= 0 ? 24 : -24, opacity: 0 }),
                    center: { x: 0, opacity: 1 },
                    exit: (d: number) => ({ x: d >= 0 ? -24 : 24, opacity: 0 }),
                }}
                initial="enter"
                animate="center"
                exit="exit"
                transition={{ duration: 0.25, ease: EASING.easeOutExpo }}
            >
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                    <AnimatePresence mode="popLayout">
                        {pagedTokens.map((token, index) => (
                            <motion.div
                                key={"oauth-" + token.id}
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{
                                    opacity: 0,
                                    scale: 0.95,
                                    transition: { duration: 0.2 }
                                }}
                                transition={{
                                    duration: 0.45,
                                    ease: EASING.easeOutExpo,
                                    delay: index === 0 ? 0 : Math.min(0.08 * Math.log2(index + 1), 0.4),
                                }}
                                layout={!searchTerm.trim()}
                            >
                                <Card token={token} />
                            </motion.div>
                        ))}
                    </AnimatePresence>
                </div>
            </motion.div>
        </AnimatePresence>
    );
}
