// ── Types ─────────────────────────────────────────────────────────

export type QuestionType = 'single' | 'scale'

export interface TestQuestionOption {
    value: string       // 'a', 'b', 'c', 'd' or '1'-'10' for scale
    label: string
    score: number        // points 0-10
}

export interface TestQuestion {
    id: string
    section: string       // section key, e.g. 'cash'
    sectionLabel: string
    category: string
    question: string
    description?: string
    type: QuestionType
    options: TestQuestionOption[]
}

export interface PersonalityResult {
    type: string
    title: string
    badge: string
    color: string
    description: string
    message: string
    traits: string[]
    strengths: string[]
    growthSuggestions: string[]
    riskAreas: string[]
    minScore: number      // inclusive
    maxScore: number      // inclusive
}

// ── Sections ──────────────────────────────────────────────────────

export const FOUNDER_TEST_SECTIONS = [
    { key: 'cash', label: 'Cash Management', questionIds: ['q1', 'q2'] },
    { key: 'planning', label: 'Planning & Forecasting', questionIds: ['q3', 'q4'] },
    { key: 'unit_econ', label: 'Unit Economics', questionIds: ['q5', 'q6'] },
    { key: 'controls', label: 'Controls & Reporting', questionIds: ['q7', 'q8'] },
    { key: 'fundraising', label: 'Fundraising Readiness', questionIds: ['q9', 'q10'] },
]

// ── Questions ─────────────────────────────────────────────────────

export const FOUNDER_TEST_QUESTIONS: TestQuestion[] = [
    // ─ Section 1: Cash & Runway ─
    {
        id: 'q1',
        section: 'cash',
        sectionLabel: 'Cash Management',
        category: 'Cash & Runway Management',
        question: 'Do you know your monthly burn rate?',
        description: 'How much cash your business spends every month',
        type: 'single',
        options: [
            { value: 'a', label: 'I track it regularly and know the exact number', score: 10 },
            { value: 'b', label: "I have a rough idea but don't review it often", score: 5 },
            { value: 'c', label: "I don't really track this", score: 0 },
        ],
    },
    {
        id: 'q2',
        section: 'cash',
        sectionLabel: 'Cash Management',
        category: 'Cash & Runway Management',
        question: 'Do you track your cash runway?',
        description: 'How many months your current cash will last',
        type: 'single',
        options: [
            { value: 'a', label: 'I update this regularly and use it for planning', score: 10 },
            { value: 'b', label: 'I check it occasionally', score: 5 },
            { value: 'c', label: "I don't track this at all", score: 0 },
        ],
    },
    // ─ Section 2: Planning & Forecasting ─
    {
        id: 'q3',
        section: 'planning',
        sectionLabel: 'Planning & Forecasting',
        category: 'Financial Planning & Forecasting',
        question: 'Do you have a structured 12+ month financial forecast?',
        type: 'single',
        options: [
            { value: 'a', label: 'Yes — a detailed financial model', score: 10 },
            { value: 'b', label: 'Rough projections in a spreadsheet', score: 5 },
            { value: 'c', label: 'No formal forecast exists', score: 0 },
        ],
    },
    {
        id: 'q4',
        section: 'planning',
        sectionLabel: 'Planning & Forecasting',
        category: 'Financial Planning & Forecasting',
        question: 'Do you compare Actual vs Plan every month?',
        description: 'Reviewing real numbers against your plan to find gaps',
        type: 'single',
        options: [
            { value: 'a', label: 'Every month — I review variances and act on them', score: 10 },
            { value: 'b', label: 'Sometimes, but not consistently', score: 5 },
            { value: 'c', label: 'Rarely or never', score: 0 },
        ],
    },
    // ─ Section 3: Unit Economics ─
    {
        id: 'q5',
        section: 'unit_econ',
        sectionLabel: 'Unit Economics',
        category: 'Unit Economics & Profitability',
        question: 'Do you know your unit economics?',
        description: 'CAC (Customer Acquisition Cost), LTV (Lifetime Value), gross margins',
        type: 'single',
        options: [
            { value: 'a', label: 'Yes — I know CAC, LTV, and gross margins precisely', score: 10 },
            { value: 'b', label: 'I know total marketing spend and revenue, roughly', score: 5 },
            { value: 'c', label: 'I have no idea about these numbers', score: 0 },
        ],
    },
    {
        id: 'q6',
        section: 'unit_econ',
        sectionLabel: 'Unit Economics',
        category: 'Unit Economics & Profitability',
        question: 'Do you know your break-even point?',
        type: 'single',
        options: [
            { value: 'a', label: 'Yes — I know break-even units and break-even month', score: 10 },
            { value: 'b', label: 'I have a rough idea', score: 5 },
            { value: 'c', label: 'Not really', score: 0 },
            { value: 'd', label: "I didn't know this was a thing", score: 0 },
        ],
    },
    // ─ Section 4: Controls & Reporting ─
    {
        id: 'q7',
        section: 'controls',
        sectionLabel: 'Controls & Reporting',
        category: 'Financial Controls & Reporting',
        question: 'Do you maintain monthly P&L, Balance Sheet, and Cash Flow statements?',
        type: 'single',
        options: [
            { value: 'a', label: 'Yes — maintained weekly or monthly', score: 10 },
            { value: 'b', label: 'We have a P&L but not the rest', score: 5 },
            { value: 'c', label: 'Our CA handles it for tax purposes only', score: 2 },
            { value: 'd', label: 'We have no financial statements', score: 0 },
        ],
    },
    {
        id: 'q8',
        section: 'controls',
        sectionLabel: 'Controls & Reporting',
        category: 'Financial Controls & Reporting',
        question: 'How thoroughly do you record transactions?',
        description: 'Scale of 1 (only big transactions) to 10 (every rupee tracked)',
        type: 'scale',
        options: Array.from({ length: 10 }, (_, i) => ({
            value: String(i + 1),
            label:
                i === 0 ? 'Only big transactions' : i === 9 ? 'Every rupee' : String(i + 1),
            score: i < 3 ? 0 : i < 6 ? 5 : 10,
        })),
    },
    // ─ Section 5: Fundraising ─
    {
        id: 'q9',
        section: 'fundraising',
        sectionLabel: 'Fundraising Readiness',
        category: 'Fundraising & Investor Readiness',
        question: 'Can you confidently explain your numbers to an investor?',
        description: 'Valuation, revenue, EBITDA, PAT, unit economics',
        type: 'single',
        options: [
            { value: 'a', label: 'Completely — I can explain everything confidently', score: 10 },
            { value: 'b', label: 'I know some of it, but need a lot of prep', score: 5 },
            { value: 'c', label: 'Stressed — significant work needed before any meeting', score: 2 },
            { value: 'd', label: 'Not ready at all', score: 0 },
        ],
    },
    {
        id: 'q10',
        section: 'fundraising',
        sectionLabel: 'Fundraising Readiness',
        category: 'Fundraising & Investor Readiness',
        question: 'Do you have an investor-ready financial model or pitch deck?',
        type: 'single',
        options: [
            { value: 'a', label: 'Yes — fully prepared', score: 10 },
            { value: 'b', label: 'Work in progress', score: 5 },
            { value: 'c', label: 'No — I need to build one', score: 0 },
        ],
    },
]

// ── Result Types (personality archetypes) ─────────────────────────
// Max possible score: 10 questions × 10 pts = 100

export const FOUNDER_TEST_RESULTS: PersonalityResult[] = [
    {
        type: 'blindfolded_founder',
        title: 'The Blindfolded Founder',
        badge: '🟥',
        color: '#DC2626',
        description:
            'Running on gut feeling with little visibility on cash, costs, or runway. High risk of sudden cash crunch.',
        message:
            "You're building something ambitious, but right now you're driving without a dashboard. The good news? This is fixable — fast. Once you start tracking cash, costs, and basic metrics, your stress will drop and your decisions will get sharper. Finance isn't here to slow you down — it's here to protect your dream.",
        traits: [
            'Runs on gut feeling',
            'Numbers feel scary or irrelevant',
            'No visibility on cash or runway',
            'High cash crunch risk',
        ],
        strengths: [
            'Bold decision-making speed',
            'Vision-driven execution',
            'Low analysis paralysis',
        ],
        growthSuggestions: [
            'Start with a simple burn-rate tracker',
            'Set up a basic monthly P&L',
            'Learn your break-even number this week',
        ],
        riskAreas: [
            'Unexpected cash shortfall',
            'Inability to raise funds',
            'No early warning system for trouble',
        ],
        minScore: 0,
        maxScore: 30,
    },
    {
        type: 'hustler_rough_numbers',
        title: 'The Hustler with Rough Numbers',
        badge: '🟠',
        color: '#D97706',
        description:
            'Knows some numbers but tracking is inconsistent. Decisions are partly data, partly instinct — reactive rather than proactive.',
        message:
            "You're doing many things right, but your finances are still running in 'jugaad mode.' With a bit more structure — forecasting, monthly reviews, and clarity on unit economics — you'll move from firefighting to actually planning your growth.",
        traits: [
            'Knows some numbers, not deeply',
            'Inconsistent tracking',
            'Partly data-driven, partly instinct',
            'Reactive with money',
        ],
        strengths: [
            'Hustle and speed',
            'Basic financial awareness',
            'Can read revenue trends',
        ],
        growthSuggestions: [
            'Set up a 90-day cash forecast',
            'Track CAC and LTV for your top segment',
            'Do a monthly Actual vs Plan review',
        ],
        riskAreas: [
            'Decisions made on incomplete data',
            'Missing growth levers hidden in numbers',
            'Investor conversations will be stressful',
        ],
        minScore: 31,
        maxScore: 55,
    },
    {
        type: 'structured_operator',
        title: 'The Structured Operator',
        badge: '🟡',
        color: '#D97706',
        description:
            'Decent systems in place. Tracks most key metrics and understands unit economics. Room to improve investor readiness.',
        message:
            "You're ahead of most founders already. Your numbers mostly make sense, and that's a big advantage. The next level is turning this into a real decision-making machine — tighter forecasts, cleaner reports, and sharper investor storytelling.",
        traits: [
            'Decent financial systems',
            'Tracks most key metrics',
            'Understands unit economics and runway',
            'Investor readiness still needs polish',
        ],
        strengths: [
            'Systematic financial thinking',
            'Can explain core metrics',
            'Monthly review discipline',
        ],
        growthSuggestions: [
            'Build a 12-month rolling forecast',
            'Create an investor-ready one-pager',
            'Deepen unit economics analysis by channel',
        ],
        riskAreas: [
            'Forecast may lack scenario planning',
            'Reporting not yet board-ready',
            'Missing narrative around the numbers',
        ],
        minScore: 56,
        maxScore: 75,
    },
    {
        type: 'finance_savvy_builder',
        title: 'The Finance-Savvy Builder',
        badge: '🟢',
        color: '#059669',
        description:
            'Good control over cash, metrics, and planning. Regular reviews and structured thinking. Investor conversations are manageable.',
        message:
            "You're running your business like a real operator, not just a dreamer. With a bit more polish in reporting, modeling, and narrative, you're very close to being fully investor-grade and scale-ready.",
        traits: [
            'Strong cash and metrics control',
            'Regular structured reviews',
            'Investor conversations manageable',
            'Thinks in systems',
        ],
        strengths: [
            'Financial discipline and consistency',
            'Strong unit economics awareness',
            'Proactive cash management',
        ],
        growthSuggestions: [
            'Stress-test your model with 3 scenarios',
            'Build a board-ready reporting pack',
            'Sharpen your fundraising narrative',
        ],
        riskAreas: [
            'May lack depth in investor storytelling',
            'Financial model may need scenario rigor',
            'Reporting cadence could be tighter',
        ],
        minScore: 76,
        maxScore: 90,
    },
    {
        type: 'investor_ready_ceo',
        title: 'The Investor-Ready CEO',
        badge: '🔵',
        color: '#1D4ED8',
        description:
            'Clear on cash, runway, unit economics, and profitability. Strong systems. Can confidently explain numbers to anyone. Ready to scale.',
        message:
            "You're playing the game at a professional level. Your numbers work for you, not against you. This puts you in a powerful position — to raise capital, scale smartly, and avoid the classic startup finance traps. Keep this discipline as you grow.",
        traits: [
            'Crystal clear on all key metrics',
            'Strong financial systems',
            'Investor-ready at any time',
            'Numbers drive every decision',
        ],
        strengths: [
            'Full-stack financial mastery',
            'Investor-grade communication',
            'Proactive risk management',
        ],
        growthSuggestions: [
            'Automate your reporting stack',
            'Build investor update templates',
            'Mentor other founders on financial discipline',
        ],
        riskAreas: [
            'Potential over-optimisation of metrics vs speed',
            'May slow decisions with too much analysis',
            'Keep the execution bias alive',
        ],
        minScore: 91,
        maxScore: 100,
    },
]

// ── Scoring Logic ─────────────────────────────────────────────────
// Returns total score, matched result type, and per-section breakdown

export interface SectionScoreResult {
    dimension: string
    label: string
    score: number
    max: number
    percentage: number
}

export const scoreAnswers = (
    answers: Record<string, string>
): {
    score: number
    resultType: string
    sectionScores: SectionScoreResult[]
} => {
    let totalScore = 0

    const sectionScores: SectionScoreResult[] = FOUNDER_TEST_SECTIONS.map((sec) => {
        let score = 0
        let max = 0

        for (const qId of sec.questionIds) {
            const question = FOUNDER_TEST_QUESTIONS.find((q) => q.id === qId)
            if (!question) continue

            const selectedValue = answers[qId]
            const option = question.options.find((o) => o.value === selectedValue)

            score += option?.score ?? 0
            max += 10
        }

        totalScore += score

        return {
            dimension: sec.key,
            label: sec.label,
            score,
            max,
            percentage: max > 0 ? Math.round((score / max) * 100) : 0,
        }
    })

    const result = FOUNDER_TEST_RESULTS.find(
        (r) => totalScore >= r.minScore && totalScore <= r.maxScore
    )

    return {
        score: totalScore,
        resultType: result?.type ?? 'structured_operator', // safe mid-tier fallback
        sectionScores,
    }
}

// ── Validate answers ──────────────────────────────────────────────

export const validateAnswers = (
    answers: Record<string, string>
): { valid: boolean; missing: string[] } => {
    const missing: string[] = []

    for (const question of FOUNDER_TEST_QUESTIONS) {
        const answer = answers[question.id]
        const validValues = question.options.map((o) => o.value)

        if (!answer || !validValues.includes(answer)) {
            missing.push(question.id)
        }
    }

    return { valid: missing.length === 0, missing }
}