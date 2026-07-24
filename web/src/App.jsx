import { useHashRoute } from "./router.js";
import { useCurrentUser } from "./currentUser.js";
import Agents from "./Agents.jsx";
import ThreadList from "./ThreadList.jsx";
import ThreadDetail from "./ThreadDetail.jsx";
import Profile from "./Profile.jsx";

export default function App() {
  const route = useHashRoute();
  const { user, error } = useCurrentUser();

  if (error) {
    return (
      <div className="login-gate">
        <p>You need to log in to use AI Rendezvous Point.</p>
        <a className="primary" href="/auth/login">
          Log in
        </a>
      </div>
    );
  }

  if (!user) {
    return <div className="empty-state">Loading…</div>;
  }

  if (route.path === "/agents") {
    if (user.role !== "admin") {
      return <ThreadList role={user.role} />;
    }
    return <Agents role={user.role} />;
  }

  const profileMatch = route.path.match(/^\/profiles\/(.+)$/);
  if (profileMatch) {
    return <Profile actorId={profileMatch[1]} role={user.role} currentActorId={user.actor_id} />;
  }

  const threadMatch = route.path.match(/^\/threads\/(.+)$/);
  if (threadMatch) {
    return <ThreadDetail threadId={threadMatch[1]} role={user.role} />;
  }

  return <ThreadList role={user.role} />;
}
