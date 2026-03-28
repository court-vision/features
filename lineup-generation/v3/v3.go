package v3

/*
v3 is a new, generalized beam search algorothm that solves the generic lineup scheduling problem
that fantasy basketball lineup optimization presents.

Since we want to generalize the algorithm to work for any scheduling problem that has a similar structure,
we assume that the proper preprocessing steps are already done (such as pre-slotting non-streamable players
into the most restrictive positions possible), leaving us with the following main inputs:

- List of open positions (we define this in a general sense, not necessarily basketball positions) for each period of time (day, week, month, etc.)
- List of items that can be scheduled (players, events, tasks, etc.)
	- Each item has a list of positions that it can be scheduled in
	- Each item has a list representing its schedule across all periods of time (NBA team schedule, etc.)
	- Optionally, each item has a score that represents its value to the schedule (fantasy points, etc.)
- List of constraints that must be satisfied (such as total number of items scheduled)
- Value function to determine how good a schedule is (such as maximizing the number of items scheduled, minimizing the number of items not scheduled, etc.)

Since the main application is for Court Vision, we also need to provide the follwing additional inputs:
- Recovery time for each item (the time it takes for an item to recover from being scheduled)
- Boolean flag to indicate if slotting has downstream effects (such as the need to slot a player into a position that is not available in the next period of time)

Beam search approach:
- Initialize a beam of size k with the initial schedule
- For each period of time, expand the beam by adding the items that can be scheduled in the open positions
- Select the k best schedules from the beam
- Repeat until the constraints are satisfied or the time limit is reached
- Return the best schedule
*/

func main() {
	
}