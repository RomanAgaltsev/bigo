# The kata corpus

Algorithm exercises reduced to their algorithms, pinned with the complexity their
author claimed when he submitted them.

## What this is for

The canonical corpus (`corpus/testdata/src`) pins *literature* bounds on textbook
algorithms. This one pins *human* claims on real submitted solutions, scored under the
**kata cost model** — the model those claims were made in, where reading input is not the
work being graded and one element comparison is one element operation.

Two questions it answers that the canonical corpus cannot:

1. **Does bigo still say what it said?** Every row carries the emitted bound, so a rule
   regression shows up as a diff.
2. **Which unsolved families actually cost the author answers?** The `top` rows are a
   ranked, measured input to that decision rather than a guess from a table of algorithm
   families.

## What is NOT here, and will never be

These fixtures are **reduced**. The originals are coursework: they carry the author's name
and cohort, links to contest submission reports, and prose restating the course's task
statements. None of that is in this repository and none of it may be added. A fixture here
contains the algorithm functions, the pins, and nothing else.

The reduction is faithful to how the claims were made, not merely convenient: the authors'
own complexity sections say *"чтение входящего массива... и вывод результата не
оценивается"* — reading the input and printing the result are not what is being graded.
Removing the I/O scaffolding removes something the claim already excluded.

If you are looking for the task statements: they belong to the course, not to this project.

## Worst case versus the accepted answer

These pins are **the author's claims**, and a claim is sometimes an average-case answer
where bigo reasons about the worst case. Quicksort is the standard example: the author
claimed `O(n log n)` for the average case while explicitly acknowledging `O(n^2)` when the
pivot choice goes badly, and bigo — if it ever bounds that function — would be right to say
`O(n^2)`.

That divergence is **by design and is not a defect on either side**. It classifies as
`loose` rather than `wrong`, because a worst-case bound dominates an average-case claim.
Read a `loose` row here as "the two are answering different questions", not automatically
as a graduation target.

## Adding a row

1. Reduce the original to its algorithm functions. Delete `main`, all I/O scaffolding, and
   every comment that is not about the algorithm.
2. Verify the reduction did not change the verdict:
   `go run ./tools/katadiff -dir <original>` against the same for the fixture.
3. Pin what the author claimed, in `//oracle:time` / `//oracle:space`, with `//oracle:source`
   naming the sprint and the claim it was transcribed from.
4. `task kata-corpus`, then read the diff before committing it.

## Exclusions

A kata with no separable algorithm function — everything inside `main` — is excluded rather
than reshaped, and listed here with the reason. A fixture rewritten until it is
bigo-friendly would pin the rewrite, not the solution.

*(None yet; this list is filled in as the corpus grows.)*
