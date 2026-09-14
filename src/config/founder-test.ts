// ── Question & Answer Types ───────────────────────────────────────

export interface TestQuestion {
    id: string
    question: string
    options: {
        value: string
        label: string
        score: number          // points this option contributes
        traits: string[]       // personality traits this option maps to
    }[]
}

export interface TestResult {
    type: string             // e.g. "Visionary", "Operator"
    title: string
    description: string
    minScore: number
    maxScore: number
}

// ── Questions ─────────────────────────────────────────────────────
// Replace these placeholders with your real questions when ready

export const FOUNDER_TEST_QUESTIONS: TestQuestion[] = [
    {
        id: 'q1',
        question: 'When starting a new project, you typically:',
        options: [
            {
                value: 'A',
                label: 'Dive in immediately and figure it out as you go',
                score: 3,
                traits: ['action-oriented', 'risk-taker'],
            },
            {
                value: 'B',
                label: 'Plan thoroughly before taking any steps',
                score: 1,
                traits: ['analytical', 'methodical'],
            },
            {
                value: 'C',
                label: 'Gather a team and delegate from the start',
                score: 2,
                traits: ['collaborative', 'leader'],
            },
            {
                value: 'D',
                label: 'Research what others have done in similar situations',
                score: 2,
                traits: ['analytical', 'strategic'],
            },
        ],
    },
    {
        id: 'q2',
        question: 'How do you handle business setbacks?',
        options: [
            {
                value: 'A',
                label: 'Pivot quickly and try a different approach',
                score: 3,
                traits: ['resilient', 'adaptive'],
            },
            {
                value: 'B',
                label: 'Analyze what went wrong before moving forward',
                score: 2,
                traits: ['analytical', 'cautious'],
            },
            {
                value: 'C',
                label: 'Seek advice from mentors or peers',
                score: 2,
                traits: ['collaborative', 'open-minded'],
            },
            {
                value: 'D',
                label: 'Stay the course — consistency beats pivoting',
                score: 1,
                traits: ['persistent', 'focused'],
            },
        ],
    },
    {
        id: 'q3',
        question: 'Your ideal business model is:',
        options: [
            {
                value: 'A',
                label: 'High risk, high reward — go big or go home',
                score: 3,
                traits: ['visionary', 'risk-taker'],
            },
            {
                value: 'B',
                label: 'Steady and profitable — slow and sustainable growth',
                score: 1,
                traits: ['operator', 'methodical'],
            },
            {
                value: 'C',
                label: 'Community driven — people over profit',
                score: 2,
                traits: ['builder', 'collaborative'],
            },
            {
                value: 'D',
                label: 'Systems first — build once, scale forever',
                score: 2,
                traits: ['strategic', 'systems-thinker'],
            },
        ],
    },
    {
        id: 'q4',
        question: 'When it comes to financial decisions:',
        options: [
            {
                value: 'A',
                label: 'Invest heavily upfront to accelerate growth',
                score: 3,
                traits: ['aggressive', 'growth-focused'],
            },
            {
                value: 'B',
                label: 'Bootstrap and reinvest profits only',
                score: 1,
                traits: ['conservative', 'independent'],
            },
            {
                value: 'C',
                label: 'Raise external capital when needed',
                score: 2,
                traits: ['strategic', 'growth-focused'],
            },
            {
                value: 'D',
                label: 'Model everything out before spending a rupee',
                score: 2,
                traits: ['analytical', 'cautious'],
            },
        ],
    },
    {
        id: 'q5',
        question: 'Your biggest strength as a founder is:',
        options: [
            {
                value: 'A',
                label: 'Selling — I can pitch anything to anyone',
                score: 3,
                traits: ['visionary', 'charismatic'],
            },
            {
                value: 'B',
                label: 'Operations — I make sure things actually get done',
                score: 1,
                traits: ['operator', 'executor'],
            },
            {
                value: 'C',
                label: 'People — I build great teams and culture',
                score: 2,
                traits: ['builder', 'leader'],
            },
            {
                value: 'D',
                label: 'Strategy — I see the chess moves others miss',
                score: 2,
                traits: ['strategic', 'analytical'],
            },
        ],
    },
]

// ── Result Types ──────────────────────────────────────────────────
// Score ranges map to founder archetypes

export const FOUNDER_TEST_RESULTS: TestResult[] = [
    {
        type: 'Operator',
        title: 'The Operator',
        description:
            'You are the backbone of any great company. You thrive on execution, systems, and making sure the wheels never stop turning. Your superpower is turning vision into reality through relentless focus and discipline.',
        minScore: 5,
        maxScore: 8,
    },
    {
        type: 'Builder',
        title: 'The Builder',
        description:
            'You lead with people and culture. You know that great teams build great products, and you invest in the humans around you. Your superpower is creating environments where everyone does their best work.',
        minScore: 9,
        maxScore: 11,
    },
    {
        type: 'Strategist',
        title: 'The Strategist',
        description:
            'You play chess while others play checkers. You see patterns, anticipate market shifts, and always have a plan B. Your superpower is turning complexity into clarity and finding opportunities others overlook.',
        minScore: 12,
        maxScore: 13,
    },
    {
        type: 'Visionary',
        title: 'The Visionary',
        description:
            'You are the spark. You dream big, move fast, and inspire everyone around you. You see the future before it exists and have the courage to chase it. Your superpower is infectious ambition.',
        minScore: 14,
        maxScore: 15,
    },
]

// ── Scoring Logic ─────────────────────────────────────────────────

export const scoreAnswers = (
    answers: Record<string, string>
): { score: number; resultType: string } => {
    let totalScore = 0

    for (const question of FOUNDER_TEST_QUESTIONS) {
        const selectedValue = answers[question.id]
        if (!selectedValue) continue

        const option = question.options.find((o) => o.value === selectedValue)
        if (option) totalScore += option.score
    }

    // find matching result type
    const result = FOUNDER_TEST_RESULTS.find(
        (r) => totalScore >= r.minScore && totalScore <= r.maxScore
    )

    return {
        score: totalScore,
        resultType: result?.type ?? 'Strategist', // default fallback
    }
}

// ── Validate answers ──────────────────────────────────────────────
// Ensures all question IDs are answered with valid option values

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