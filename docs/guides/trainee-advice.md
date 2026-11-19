# Advice for Trainees

This page is advice from Erick to trainees on working with agents. It is one view, not a rule.

## Build up slowly

1. Write code yourself.
2. Write code interactively with an agent.
3. Write issues, let agents turn them into PRs, and read each PR carefully.
4. Adopt an agent fleet only when you have high confidence in step 3.

Each step builds your taste and your confidence. Taste comes from reading the code. Confidence comes from checking the results.

[Where to Start](where-to-start.md) gives the tooling for steps 2 to 4.

## Start with the starter skills, and kaizen

Start from [skills-starter](https://github.com/matsengrp/skills-starter). Its key skill is `/kaizen`. When an agent does something wrong, ask whether a small, general change would prevent that class of mistake. Then commit that change to a skill or an `AGENTS.md`. The agent drafts the change; you decide whether it lands. Don't add a rule for every mistake.

## Agent work is your work

You are responsible for what your agents produce. Before you share a result, you must know that you can stand by it. Read the code and check the numbers.

## Trust takes time to build

Trainees describe David Baker as someone who spends his days moving around the lab. He talks to people and connects their ideas. He probably does not review the code in his lab. That is one picture of where you might want to go with agents.

It also shows how long the path is. Baker built his lab over decades. He built a system that he trusts to produce rigorous science. You need to build the same trust in your agents, and agents are not a shortcut to it.

I have coded for decades and advised people for over 15 years. Over that time I learned when I can start to trust a trainee or a staff member, and when I must probe deeply. You will build the same sense for your agents, one checked result at a time.

## Prototype fast, finalize carefully

Ask yourself which stage a piece of work is in. Be fast where it is cheap to be wrong, and careful where it isn't. Build a prototype fast to learn whether an idea works. Invest in the code only after that. To finalize can mean that you throw away all of the prototype and start again from the right idea.

## Automate operations, keep the science

Decide what is operations and what is science. Operations include cluster checks, rebases, and waiting for CI. Automate them first. The science is yours: the questions, the methods, and what a result means. Some technical or mathematical code is too hard for the agents. Write that code yourself.

## Work with friends

Present your results to your friends. Can you explain your analysis to them? Then explain it to your PI. Each talk tests whether you understand your own work. Agent work alone can go deep fast, and it gets hard to explain. Talk to people early.

## Let fun and shame guide you

Let fun be your guide: work on what you enjoy. Let shame be your guide too: if you would not want to show a result to a friend, fix it before you build on it.

## If you are frustrated

Frustration is OK. It usually means one of two things. You may need to improve your agent skills: use `/kaizen`, and ask a friend how they work. Or the task may be at the limit of what the models can do. In that case, congratulations: try again in six months.

## A counterpoint

Jascha Sohl-Dickstein (Anthropic) gives the opposite advice on pace in his talk ["Advice for a (young) investigator in the first and last days of the Anthropocene"](https://docs.google.com/presentation/d/1qVFDW8qT4CC4E_2TSVevrDbZ_Z9Utu_I1z0-ISLwZts/). One slide says: "Learn the new tools. Benefit from the exponential. They are often non-ergonomic. Use them anyway." His speaker notes add: "you can't fall behind the exponential if you are on the exponential." Read both views and decide for yourself.

## See also

- [Driving Science](driving-science.md): how the full system works.
