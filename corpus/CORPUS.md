# bigo canonical algorithm corpus — oracle golden

GENERATED — do not edit; regenerate with `task corpus`.

Literature-pinned worst-case bounds vs unaided inference. `exact` = inference
matches the literature; `loose` = sound but imprecise (a graduation target);
`top` = unverifiable (the annotate-or-trust evidence rows). A `wrong` never
appears here: it fails the build. Algorithms considered and kept out are in
[EXCLUSIONS.md](EXCLUSIONS.md). **This is not a coverage metric** — read
composition, not a percentage.

**Entries: 15**

## Time statuses

| Status | Count |
|---|---|
| exact | 5 |
| loose | 0 |
| top | 10 |

## Space statuses (pinned entries only)

| Status | Count |
|---|---|
| exact | 9 |
| loose | 1 |
| top | 5 |

## Per family

| Family | Entries |
|---|---|
| searching | 5 |
| sorting | 10 |

## Entries

| Function | Time pin | Time got | Status | Space pin | Space got | Status | Cause | Source |
|---|---|---|---|---|---|---|---|---|
| searching.BinarySearch | O(log(len(s))) | unverifiable | top | O(1) | O(1) | exact | loop | CLRS ex. 2.3-5; en.wikipedia.org/wiki/Binary_search_algorithm |
| searching.BinarySearchRec | O(log(len(s))) | O(log(len(s))) | exact | O(log(len(s))) | O(log(len(s))) | exact |  | CLRS ex. 2.3-5 (recursive form) |
| searching.FirstOccurrence | O(log(len(s))) | unverifiable | top | O(1) | O(1) | exact | loop | en.wikipedia.org/wiki/Binary_search_algorithm (leftmost variant) |
| searching.LinearSearch | O(len(s)) | O(len(s)) | exact | O(1) | O(1) | exact |  | en.wikipedia.org/wiki/Linear_search |
| searching.SearchRotated | O(log(len(s))) | unverifiable | top | O(1) | O(1) | exact | loop | www.geeksforgeeks.org/search-an-element-in-a-sorted-and-pivoted-array/ (bound reference) |
| sorting.BubbleSort | O(len(s)^2) | O(len(s)^2) | exact | O(1) | O(1) | exact |  | CLRS problem 2-2; en.wikipedia.org/wiki/Bubble_sort |
| sorting.BucketSort | O(len(s)^2) | unverifiable | top | O(len(s)) | unverifiable | top | loop | CLRS §8.4 (worst case); en.wikipedia.org/wiki/Bucket_sort |
| sorting.CountingSort | O(k + len(s)) | unverifiable | top | O(k + len(s)) | unverifiable | top | loop | CLRS §8.2; en.wikipedia.org/wiki/Counting_sort |
| sorting.HeapSort | O(len(s) log(len(s))) | unverifiable | top | O(1) | O(len(s)) | loose | call | CLRS §6.4; en.wikipedia.org/wiki/Heapsort |
| sorting.InsertionSort | O(len(s)^2) | O(len(s)^2) | exact | O(1) | O(1) | exact |  | CLRS §2.1; en.wikipedia.org/wiki/Insertion_sort (worst case) |
| sorting.MergeSort | O(len(s) log(len(s))) | unverifiable | top | O(len(s)) | unverifiable | top | call | CLRS §2.3.1; en.wikipedia.org/wiki/Merge_sort |
| sorting.QuickSort | O(len(s)^2) | unverifiable | top | O(len(s)) | unverifiable | top | call | CLRS §7; en.wikipedia.org/wiki/Quicksort (worst case) |
| sorting.RadixSortLSD | O(len(s)) | unverifiable | top | O(len(s)) | unverifiable | top | loop | CLRS §8.3 (fixed d, k); en.wikipedia.org/wiki/Radix_sort |
| sorting.SelectionSort | O(len(s)^2) | O(len(s)^2) | exact | O(1) | O(1) | exact |  | CLRS ex. 2.2-2; en.wikipedia.org/wiki/Selection_sort |
| sorting.ShellSort | O(len(s)^2) | unverifiable | top | O(1) | O(1) | exact | loop | en.wikipedia.org/wiki/Shellsort (Shell's sequence, worst case) |
